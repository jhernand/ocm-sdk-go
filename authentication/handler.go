/*
Copyright (c) 2019 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package authentication

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/ghodss/yaml"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/errors"
)

// HandlerBuilder contains the data and logic needed to create a new authentication handler. Don't
// create objects of this type directly, use the NewHandler function instead.
type HandlerBuilder struct {
	logger       sdk.Logger
	publicPaths  []string
	keysFiles    []string
	keysURLs     []string
	keysCAs      *x509.CertPool
	keysInsecure bool
	aclFiles     []string
	next         http.Handler
}

// Handler is an HTTP handler that checks authentication using the JWT tokens from the authorization
// header.
type Handler struct {
	logger        sdk.Logger
	publicPaths   []*regexp.Regexp
	tokenParser   *jwt.Parser
	keysFiles     []string
	keysURLs      []string
	keysClient    *http.Client
	keys          *sync.Map
	lastKeyReload time.Time
	aclItems      map[string]*regexp.Regexp
	next          http.Handler
}

// NewHandler creates a builder that can then be configured and used to create authentication
// handlers.
func NewHandler() *HandlerBuilder {
	return &HandlerBuilder{}
}

// Logger sets the logger that the middleware will use to send messages to the log. This is
// mandatory.
func (b *HandlerBuilder) Logger(value sdk.Logger) *HandlerBuilder {
	b.logger = value
	return b
}

// Public sets a regular expression that defines the parts of the URL space that considered public,
// and therefore require no authentication. This method may be called multiple times and then all
// the given regular expressions will be used to check what parts of the URL space are public.
func (b *HandlerBuilder) Public(value string) *HandlerBuilder {
	b.publicPaths = append(b.publicPaths, value)
	return b
}

// KeysFile sets the location of a file containing a JSON web key set that will be used to verify
// the signatures of the tokens. The keys from this file will be loaded when a token is received
// containing an unknown key identifier.
//
// At least one keys file or one keys URL is mandatory.
func (b *HandlerBuilder) KeysFile(value string) *HandlerBuilder {
	if value != "" {
		b.keysFiles = append(b.keysFiles, value)
	}
	return b
}

// KeysURL sets the URL containing a JSON web key set that will be used to verify the signatures of
// the tokens. The keys from these URLs will be loaded when a token is received containing an
// unknown key identifier.
//
// At least one keys file or one keys URL is mandatory.
func (b *HandlerBuilder) KeysURL(value string) *HandlerBuilder {
	if value != "" {
		b.keysURLs = append(b.keysURLs, value)
	}
	return b
}

// KeysCAs sets the certificate authorities that will be trusted when verifying the certificate of
// the web server where keys are loaded from.
func (b *HandlerBuilder) KeysCAs(value *x509.CertPool) *HandlerBuilder {
	b.keysCAs = value
	return b
}

// KeysInsecure sets the flag that indicates that the certificate of the web server where the keys
// are loaded from should not be checked. The default is false and changing it to true makes the
// token verification insecure, so refrain from doing that in security sensitive environments.
func (b *HandlerBuilder) KeysInsecure(value bool) *HandlerBuilder {
	b.keysInsecure = value
	return b
}

// ACLFile sets a file that contains items of the access control list. This should be a YAML file
// with the following format:
//
// - claim: email
//   pattern: ^.*@redhat\.com$
//
// - claim: sub
//   pattern: ^f:b3f7b485-7184-43c8-8169-37bd6d1fe4aa:myuser$
//
// The claim field is the name of the claim of the JWT token that will be checked. The pattern field
// is a regular expression. If the claim matches the regular expression then access will be allowed.
//
// If the ACL is empty then access will be allowed to all JWT tokens.
//
// If the ACL has at least one item then access will be allowed only to tokens that match at least
// one of the items.
func (b *HandlerBuilder) ACLFile(value string) *HandlerBuilder {
	if value != "" {
		b.aclFiles = append(b.aclFiles, value)
	}
	return b
}

// Next sets the HTTP handler that will be called when the authentication handler has authenticated
// correctly the request. This is mandatory.
func (b *HandlerBuilder) Next(value http.Handler) *HandlerBuilder {
	b.next = value
	return b
}

// Build uses the data stored in the builder to create a new authentication handler.
func (b *HandlerBuilder) Build() (handler *Handler, err error) {
	// Check parameters:
	if b.logger == nil {
		err = fmt.Errorf("logger is mandatory")
		return
	}
	if b.next == nil {
		err = fmt.Errorf("next handler is mandatory")
		return
	}

	// Check that there is at least one keys source:
	if len(b.keysFiles)+len(b.keysURLs) == 0 {
		err = fmt.Errorf("at least one keys file or one keys URL must be configured")
		return
	}

	// Check that all the configured keys files exist:
	for _, file := range b.keysFiles {
		var info os.FileInfo
		info, err = os.Stat(file)
		if err != nil {
			err = fmt.Errorf("keys file '%s' doesn't exist: %v", file, err)
			return
		}
		if !info.Mode().IsRegular() {
			err = fmt.Errorf("keys file '%s' isn't a regular file", file)
			return
		}
	}

	// Check that all the configured keys URLs are valid HTTPS URLs:
	for _, addr := range b.keysURLs {
		var parsed *url.URL
		parsed, err = url.Parse(addr)
		if err != nil {
			err = fmt.Errorf("keys URL '%s' isn't a valid URL: %v", addr, err)
			return
		}
		if !strings.EqualFold(parsed.Scheme, "https") {
			err = fmt.Errorf(
				"keys URL '%s' doesn't use the HTTPS protocol: %v",
				addr, err,
			)
		}
	}

	// Create the HTTP client that will be used to load the keys:
	keysClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            b.keysCAs,
				InsecureSkipVerify: b.keysInsecure, // nolint
			},
		},
	}

	// Try to compile the regular expressions that define the parts of the URL space that are
	// public:
	public := make([]*regexp.Regexp, len(b.publicPaths))
	for i, expr := range b.publicPaths {
		public[i], err = regexp.Compile(expr)
		if err != nil {
			return
		}
	}

	// Create the bearer token parser:
	tokenParser := &jwt.Parser{}

	// Make copies of the lists of keys files and URLs:
	keysFiles := make([]string, len(b.keysFiles))
	copy(keysFiles, b.keysFiles)
	keysURLs := make([]string, len(b.keysURLs))
	copy(keysURLs, b.keysURLs)

	// Create the initial empty map of keys:
	keys := &sync.Map{}

	// Load the ACL files:
	aclItems := map[string]*regexp.Regexp{}
	for _, file := range b.aclFiles {
		err = b.loadACLFile(file, aclItems)
		if err != nil {
			return
		}
	}

	// Create and populate the object:
	handler = &Handler{
		logger:      b.logger,
		publicPaths: public,
		tokenParser: tokenParser,
		keysFiles:   keysFiles,
		keysURLs:    keysURLs,
		keysClient:  keysClient,
		keys:        keys,
		aclItems:    aclItems,
		next:        b.next,
	}

	return
}

// aclItem is the type used to read a single ACL item from a YAML document.
type aclItem struct {
	Claim   string `json:"claim"`
	Pattern string `json:"pattern"`
}

// loadACLFile loads the given ACL file into the given map of ACL items.
func (b *HandlerBuilder) loadACLFile(file string, items map[string]*regexp.Regexp) error {
	// Load the YAML data:
	yamlData, err := ioutil.ReadFile(file) // nolint
	if err != nil {
		return err
	}

	// Parse the YAML data:
	var listData []aclItem
	err = yaml.Unmarshal(yamlData, &listData)
	if err != nil {
		return err
	}

	// Process the items:
	for _, itemData := range listData {
		items[itemData.Claim], err = regexp.Compile(itemData.Pattern)
		if err != nil {
			return err
		}
	}

	return nil
}

// ServeHTTP is the implementation of the HTTP handler interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the context:
	ctx := r.Context()

	// Check if the requested path is public, and skip authentication if it is:
	for _, expr := range h.publicPaths {
		if expr.MatchString(r.URL.Path) {
			h.next.ServeHTTP(w, r)
			return
		}
	}

	// Extract the bearer token from the authorization header:
	header := r.Header.Get("Authorization")
	if header == "" {
		h.sendError(
			w, r,
			"Request doesn't contain the 'Authorization' header",
		)
		return
	}
	matches := bearerRE.FindStringSubmatch(header)
	if len(matches) != 3 {
		h.sendError(
			w, r,
			"Authorization header '%s' is malformed",
			header,
		)
		return
	}
	typ := matches[1]
	bearer := matches[2]
	if !strings.EqualFold(typ, "Bearer") {
		h.sendError(
			w, r,
			"Authentication type '%s' isn't supported",
			typ,
		)
		return
	}

	// Use the JWT library to verify that the token is correctly signed and that the basic
	// claims are correct:
	token, claims, ok := h.checkToken(w, r, bearer)
	if !ok {
		return
	}

	// The library that we use considers tokens valid if the claims that it checks don't exist,
	// but we want to reject those tokens, so we need to do some additional validations:
	ok = h.checkClaims(w, r, claims)
	if !ok {
		return
	}

	// Check if the claims match at least one of the ACL items:
	ok = h.checkACL(w, r, claims)
	if !ok {
		return
	}

	// Add the token to the context:
	ctx = ContextWithToken(ctx, token)
	r = r.WithContext(ctx)

	// Call the next handler:
	h.next.ServeHTTP(w, r)
}

// selectKey selects the key that should be used to verify the given token.
func (h *Handler) selectKey(ctx context.Context, token *jwt.Token) (key interface{}, err error) {
	// Get the key identifier:
	value, ok := token.Header["kid"]
	if !ok {
		err = fmt.Errorf("token doesn't have a 'kid' field in the header")
		return
	}
	kid, ok := value.(string)
	if !ok {
		err = fmt.Errorf(
			"token has a 'kid' field, but it is a %T instead of a string",
			value,
		)
		return
	}

	// Get the key for that key identifier. If there is no such key and we didn't reload keys
	// recently then we try to reload them now.
	key, ok = h.keys.Load(kid)
	if !ok && time.Since(h.lastKeyReload) > 1*time.Minute {
		err = h.loadKeys(ctx)
		if err != nil {
			return
		}
		h.lastKeyReload = time.Now()
		key, ok = h.keys.Load(kid)
	}
	if !ok {
		err = fmt.Errorf("there is no key for key identifier '%s'", kid)
		return
	}

	return
}

// keyData is the type used to read a single key from a JSON document.
type keyData struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// setData is the type used to read a collection of keys from a JSON document.
type setData struct {
	Keys []keyData `json:"keys"`
}

// loadKeys loads the JSON web key set from the URLs specified in the configuration.
func (h *Handler) loadKeys(ctx context.Context) error {
	// Load keys from the files given in the configuration:
	for _, keysFile := range h.keysFiles {
		h.logger.Info(ctx, "Loading keys from file '%s'", keysFile)
		err := h.loadKeysFile(ctx, keysFile)
		if err != nil {
			h.logger.Error(ctx, "Can't load keys from file '%s': %v", keysFile, err)
		}
	}

	// Load keys from URLs given in the configuration:
	for _, keysURL := range h.keysURLs {
		h.logger.Info(ctx, "Loading keys from URL '%s'", keysURL)
		err := h.loadKeysURL(ctx, keysURL)
		if err != nil {
			h.logger.Error(ctx, "Can't load keys from URL '%s': %v", keysURL, err)
		}
	}

	return nil
}

// loadKeysFile loads a JSON we key set from a file.
func (h *Handler) loadKeysFile(ctx context.Context, file string) error {
	reader, err := os.Open(file) // nolint
	if err != nil {
		return err
	}
	return h.readKeys(ctx, reader)
}

// loadKeysURL loads a JSON we key set from an URL.
func (h *Handler) loadKeysURL(ctx context.Context, addr string) error {
	request, err := http.NewRequest(http.MethodGet, addr, nil)
	if err != nil {
		return err
	}
	request = request.WithContext(ctx)
	response, err := h.keysClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			h.logger.Error(
				ctx,
				"Can't close response body for request to '%s': %v",
				addr, err,
			)
		}
	}()
	return h.readKeys(ctx, response.Body)
}

// readKeys reads the keys from JSON web key set available in the given reader.
func (h *Handler) readKeys(ctx context.Context, reader io.Reader) error {
	// Read the JSON data:
	jsonData, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	// Parse the JSON data:
	var setData setData
	err = json.Unmarshal(jsonData, &setData)
	if err != nil {
		return err
	}

	// Convert the key data to actual keys that can be used to verify the signatures of the
	// tokens:
	for _, keyData := range setData.Keys {
		if h.logger.DebugEnabled() {
			h.logger.Debug(ctx, "Value of 'kid' is '%s'", keyData.Kid)
			h.logger.Debug(ctx, "Value of 'kty' is '%s'", keyData.Kty)
			h.logger.Debug(ctx, "Value of 'alg' is '%s'", keyData.Alg)
			h.logger.Debug(ctx, "Value of 'e' is '%s'", keyData.E)
			h.logger.Debug(ctx, "Value of 'n' is '%s'", keyData.N)
		}
		if keyData.Kid == "" {
			h.logger.Error(ctx, "Can't read key because 'kid' is empty")
			continue
		}
		if keyData.Kty == "" {
			h.logger.Error(
				ctx,
				"Can't read key '%s' because 'kty' is empty",
				keyData.Kid,
			)
			continue
		}
		if keyData.Alg == "" {
			h.logger.Error(
				ctx,
				"Can't read key '%s' because 'alg' is empty",
				keyData.Kid,
			)
			continue
		}
		if keyData.E == "" {
			h.logger.Error(
				ctx,
				"Can't read key '%s' because 'e' is empty",
				keyData.Kid,
			)
			continue
		}
		if keyData.E == "" {
			h.logger.Error(
				ctx,
				"Can't read key '%s' because 'n' is empty",
				keyData.Kid,
			)
			continue
		}
		var key interface{}
		key, err = h.parseKey(keyData)
		if err != nil {
			h.logger.Error(
				ctx,
				"Key '%s' will be ignored because it can't be parsed",
				keyData.Kid,
			)
			continue
		}
		h.keys.Store(keyData.Kid, key)
		h.logger.Info(ctx, "Loaded key '%s'", keyData.Kid)
	}

	return nil
}

// parseKey converts the key data loaded from the JSON document to an actual key that can be used
// to verify the signatures of tokens.
func (h *Handler) parseKey(data keyData) (key interface{}, err error) {
	// Check key type:
	if data.Kty != "RSA" {
		err = fmt.Errorf("key type '%s' isn't supported", data.Kty)
		return
	}

	// Decode the e and n values:
	nb, err := base64.RawURLEncoding.DecodeString(data.N)
	if err != nil {
		return
	}
	eb, err := base64.RawURLEncoding.DecodeString(data.E)
	if err != nil {
		return
	}

	// Create the key:
	key = &rsa.PublicKey{
		N: new(big.Int).SetBytes(nb),
		E: int(new(big.Int).SetBytes(eb).Int64()),
	}

	return
}

// checkToken checks if the token is valid. If it is valid it returns the parsed token, the
// claims and true. If it isn't valid it sends an error response to the client and returns false.
func (h *Handler) checkToken(w http.ResponseWriter, r *http.Request,
	bearer string) (token *jwt.Token, claims jwt.MapClaims, ok bool) {
	// Get the context:
	ctx := r.Context()

	// Parse the token:
	claims = jwt.MapClaims{}
	token, err := h.tokenParser.ParseWithClaims(
		bearer, claims,
		func(token *jwt.Token) (key interface{}, err error) {
			return h.selectKey(ctx, token)
		},
	)
	if err != nil {
		switch typed := err.(type) {
		case *jwt.ValidationError:
			switch {
			case typed.Errors&jwt.ValidationErrorMalformed != 0:
				h.sendError(
					w, r,
					"Bearer token is malformed",
				)
			case typed.Errors&jwt.ValidationErrorUnverifiable != 0:
				h.sendError(
					w, r,
					"Bearer token can't be verified",
				)
			case typed.Errors&jwt.ValidationErrorSignatureInvalid != 0:
				h.sendError(
					w, r,
					"Signature of bearer token isn't valid",
				)
			case typed.Errors&jwt.ValidationErrorExpired != 0:
				h.sendError(
					w, r,
					"Bearer token is expired",
				)
			case typed.Errors&jwt.ValidationErrorIssuedAt != 0:
				h.sendError(
					w, r,
					"Bearer token was issued in the future",
				)
			case typed.Errors&jwt.ValidationErrorNotValidYet != 0:
				h.sendError(
					w, r,
					"Bearer token isn't valid yet",
				)
			default:
				h.sendError(
					w, r,
					"Bearer token isn't valid",
				)
			}
		default:
			h.sendError(
				w, r,
				"Bearer token is malformed",
			)
		}
		ok = false
		return
	}
	ok = true
	return
}

// checkClaims checks that the required claims are present and that they have valid values. If
// something is wrong it sends an error response to the client and returns false.
func (h *Handler) checkClaims(w http.ResponseWriter, r *http.Request,
	claims jwt.MapClaims) bool {
	value, ok := h.checkStringClaim(w, r, claims, "typ")
	if !ok {
		return false
	}
	if !strings.EqualFold(value, "Bearer") {
		h.sendError(
			w, r,
			"Bearer token type '%s' isn't supported",
			value,
		)
		return false
	}
	_, ok = h.checkTimeClaim(w, r, claims, "iat")
	if !ok {
		return false
	}
	_, ok = h.checkTimeClaim(w, r, claims, "exp")
	if !ok {
		return false
	}
	return true
}

// checkTimeClaim checks that the given claim exists and that the value is a time. If it doesn't
// exist or it has a wrong type it sends an error response to the client and returns false. If it
// exists it returns its value and true.
func (h *Handler) checkTimeClaim(w http.ResponseWriter, r *http.Request,
	claims jwt.MapClaims, name string) (result time.Time, ok bool) {
	value, ok := h.checkClaim(w, r, claims, name)
	if !ok {
		return
	}
	seconds, ok := value.(float64)
	if !ok {
		h.sendError(
			w, r,
			"Bearer token claim '%s' contains incorrect time value '%v'",
			name, value,
		)
		return
	}
	result = time.Unix(int64(seconds), 0)
	return
}

// checkStringClaim checks that the given claim exists and that the value is a string. If it doesn't
// exist or it has a wrong type it sends an error response to the client and returns false. If it
// exists it returns its value and true.
func (h *Handler) checkStringClaim(w http.ResponseWriter, r *http.Request,
	claims jwt.MapClaims, name string) (result string, ok bool) {
	value, ok := h.checkClaim(w, r, claims, name)
	if !ok {
		return
	}
	result, ok = value.(string)
	if !ok {
		h.sendError(
			w, r,
			"Bearer token claim '%s' contains incorrect text value '%v'",
			name, value,
		)
		return
	}
	return
}

// checkClaim checks that the given claim exists. If it doesn't exist it sends an error response to
// the client and returns false. If it exists it returns its value and true.
func (h *Handler) checkClaim(w http.ResponseWriter, r *http.Request, claims jwt.MapClaims,
	name string) (value interface{}, ok bool) {
	value, ok = claims[name]
	if !ok {
		h.sendError(
			w, r,
			"Bearer token doesn't contain required claim '%s'",
			name,
		)
		return
	}
	return
}

// checkACL checks if the given set of claims match at least one of the items of the access control
// list. If there is no match it sends an error response to the client and returns false. If there
// is a match or the ACL is empty it returns true.
func (h *Handler) checkACL(w http.ResponseWriter, r *http.Request, claims jwt.MapClaims) bool {
	// If there are no ACL items we consider that there are no restrictions, therefore we
	// return true immediately:
	if len(h.aclItems) == 0 {
		return true
	}

	// Check all the ACL items:
	for claim, pattern := range h.aclItems {
		value, ok := claims[claim]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		if pattern.MatchString(text) {
			return true
		}
	}

	// No match, so the access is denied:
	h.sendError(
		w, r,
		"Access denied",
	)
	return false
}

// sendError sends an error response to the client with the given status code and with a message
// composed using the given format and arguments as the fmt.Sprintf function does.
func (h *Handler) sendError(w http.ResponseWriter, r *http.Request, format string, args ...interface{}) {
	// Get the context:
	ctx := r.Context()

	// Prepare the body:
	segments := strings.Split(r.URL.Path, "/")
	realm := ""
	builder := errors.NewError()
	builder.ID(fmt.Sprintf("%d", http.StatusUnauthorized))
	if len(segments) >= 4 {
		prefix := segments[1]
		service := segments[2]
		version := segments[3]
		builder.HREF(fmt.Sprintf(
			"/%s/%s/%s/errors/%d",
			prefix, service, version, http.StatusUnauthorized,
		))
		builder.Code(fmt.Sprintf(
			"%s-%d",
			strings.ToUpper(strings.ReplaceAll(service, "_", "-")),
			http.StatusUnauthorized,
		))
		realm = fmt.Sprintf("%s/%s", service, version)
	}
	builder.Reason(fmt.Sprintf(format, args...))
	body, err := builder.Build()
	if err != nil {
		h.logger.Error(ctx, "Can't build error response: %v", err)
		errors.SendPanic(w, r)
	}

	// Send the response:
	w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=\"%s\"", realm))
	errors.SendError(w, r, body)
}

// Regular expression used to extract the bearer token from the authorization header:
var bearerRE = regexp.MustCompile(`^([a-zA-Z0-9]+)\s+(.*)$`)
