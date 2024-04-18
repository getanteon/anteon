/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

package types

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	validator "github.com/asaskevich/govalidator"
	"go.ddosify.com/ddosify/core/util"
)

// Constants for Scenario field values
const (
	// Constants of the Protocol types
	ProtocolHTTP  = "HTTP"
	ProtocolHTTPS = "HTTPS"

	// Constants of the Auth types
	AuthHttpBasic = "basic"

	// Max sleep in ms (90s)
	maxSleep = 90000

	// Should match environment variables, reference
	EnvironmentVariableRegexStr = `{{[a-zA-Z$][a-zA-Z0-9_().-]*}}`

	// Should match environment variables, definition, exact match
	EnvironmentVariableNameStr = `^[a-zA-Z][a-zA-Z0-9_-]*$`
)

// SupportedProtocols should be updated whenever a new requester.Requester interface implemented
var SupportedProtocols = [...]string{ProtocolHTTP, ProtocolHTTPS}
var supportedProtocolMethods = []string{
	http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
	http.MethodPatch, http.MethodHead, http.MethodOptions,
}
var supportedAuthentications = []string{
	AuthHttpBasic,
}

var envVarRegexp *regexp.Regexp
var envVarNameRegexp *regexp.Regexp

func init() {
	envVarRegexp = regexp.MustCompile(EnvironmentVariableRegexStr)
	envVarNameRegexp = regexp.MustCompile(EnvironmentVariableNameStr)
}

// Scenario struct contains a list of ScenarioStep so scenario.ScenarioService can execute the scenario step by step.
type Scenario struct {
	Steps   []ScenarioStep
	Envs    map[string]interface{}
	CsvVars []string           // only for validation
	Data    map[string]CsvData // populated data
}

func (s *Scenario) validate() error {
	stepIds := make(map[uint16]struct{}, len(s.Steps))
	definedEnvs := map[string]struct{}{}

	// add global envs
	for key := range s.Envs {
		if !envVarNameRegexp.MatchString(key) { // not a valid env definition
			return fmt.Errorf("env key is not valid: %s", key)
		}
		definedEnvs[key] = struct{}{} // exist
	}
	// add csv vars
	for _, key := range s.CsvVars { // data.info.name
		splitted := strings.Split(key, ".")
		if len(splitted) > 3 {
			return fmt.Errorf("csv key can not have dot in it: %s", key)
		}
		for _, s := range splitted {
			if !envVarNameRegexp.MatchString(s) { // not a valid env definition
				return fmt.Errorf("csv key is not valid: %s", key)
			}
		}
		definedEnvs[key] = struct{}{} // exist
	}

	for _, st := range s.Steps {
		if err := st.validate(definedEnvs); err != nil {
			return err
		}

		// enrich Envs map with captured envs from each step
		for _, ce := range st.EnvsToCapture {
			if !envVarNameRegexp.MatchString(ce.Name) { // not a valid env definition
				return fmt.Errorf("captured env key is not valid: %s", ce.Name)
			}
			definedEnvs[ce.Name] = struct{}{}
		}

		if _, ok := stepIds[st.ID]; ok {
			return fmt.Errorf("duplicate step id: %d", st.ID)
		}
		stepIds[st.ID] = struct{}{}
	}
	return nil
}

func checkEnvsValidInStep(st *ScenarioStep, definedEnvs map[string]struct{}) error {
	var err error
	matchInEnvs := func(matches []string) error {
		for _, v := range matches {
			if _, ok := definedEnvs[v[2:len(v)-2]]; !ok { // {{....}}
				// utility functions are matched too, check if starts with rand
				// TODO: find a better solution about utility functions and validation checks

				if strings.HasPrefix(v[2:len(v)-2], "rand(") {
					if _, ok := definedEnvs[v[7:len(v)-3]]; ok {
						continue
					}
				}

				if strings.HasPrefix(v[2:len(v)-2], "$") {
					varName := v[3 : len(v)-2]
					if _, ok := os.LookupEnv(varName); ok {
						continue
					}

					return EnvironmentNotDefinedError{
						msg: fmt.Sprintf("%s is not found in the operating system environment variables", v),
					}
				}

				return EnvironmentNotDefinedError{
					msg: fmt.Sprintf("%s is not defined to use by global and captured environments", v),
				}
			}
		}
		return nil
	}

	f := func(source string) error {
		matches := envVarRegexp.FindAllString(source, -1)
		return matchInEnvs(matches)
	}

	// check env usage in url
	err = f(st.URL)
	if err != nil {
		return err
	}

	// check env usage in header
	for k, v := range st.Headers {
		err = f(k)
		if err != nil {
			return err
		}

		err = f(v)
		if err != nil {
			return err
		}
	}

	// check env usage in payload
	err = f(st.Payload)
	return err

}

// ScenarioStep represents one step of a Scenario.
// This struct should be able to include all necessary data in a network packet for SupportedProtocols.
type ScenarioStep struct {
	// ID of the Item. Should be given by the client.
	ID uint16

	// Name of the Item.
	Name string

	// Request Method
	Method string

	// Authentication
	Auth Auth

	// A TLS cert
	Cert tls.Certificate

	// A TLS cert pool
	CertPool *x509.CertPool

	// Request Headers
	Headers map[string]string

	// Request payload
	Payload string

	// Target URL
	URL string

	// Connection timeout duration of the request in seconds
	Timeout int

	// Sleep duration after running the step. Can be a time range like "300-500" or an exact duration like "350" in ms
	Sleep string

	// Protocol spesific request parameters. For ex: DisableRedirects:true for Http requests
	Custom map[string]interface{}

	// Envs to capture from response of this step
	EnvsToCapture []EnvCaptureConf

	// assertion expressions
	Assertions []string
}

type SourceType string

const (
	Header SourceType = "header"
	Body   SourceType = "body"
	Cookie SourceType = "cookies"
)

type RegexCaptureConf struct {
	Exp *string `json:"exp"`
	No  int     `json:"matchNo"`
}

type EnvCaptureConf struct {
	JsonPath   *string           `json:"json_path"`
	Xpath      *string           `json:"xpath"`
	XpathHtml  *string           `json:"xpath_html"`
	RegExp     *RegexCaptureConf `json:"regexp"`
	Name       string            `json:"as"`
	From       SourceType        `json:"from"`
	Key        *string           `json:"header_key"`
	CookieName *string           `json:"cookie_name"`
}

type CsvData struct {
	Rows   []map[string]interface{}
	Random bool
}

// Auth struct should be able to include all necessary authentication realated data for supportedAuthentications.
type Auth struct {
	Type     string
	Username string
	Password string
}

func (si *ScenarioStep) validate(definedEnvs map[string]struct{}) error {
	if !util.StringInSlice(si.Method, supportedProtocolMethods) {
		return fmt.Errorf("unsupported Request Method: %s", si.Method)
	}
	if si.Auth != (Auth{}) && !util.StringInSlice(si.Auth.Type, supportedAuthentications) {
		return fmt.Errorf("unsupported Authentication Method (%s) ", si.Auth.Type)
	}
	if si.ID == 0 {
		return fmt.Errorf("step ID should be greater than zero")
	}
	if !envVarRegexp.MatchString(si.URL) && !validator.IsURL(strings.ReplaceAll(si.URL, " ", "_")) {
		return fmt.Errorf("target is not valid: %s", si.URL)
	}
	if si.Sleep != "" {
		sleep := strings.Split(si.Sleep, "-")

		// Avoid invalid syntax like "-300-500"
		if len(sleep) > 2 {
			return fmt.Errorf("sleep expression is not valid: %s", si.Sleep)
		}

		// Validate string to int conversion
		for _, s := range sleep {
			dur, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("sleep is not valid: %s", si.Sleep)
			}

			if dur > maxSleep {
				return fmt.Errorf("maximum sleep limit exceeded. provided: %d ms, maximum: %d ms", dur, maxSleep)
			}
		}
	}

	for _, conf := range si.EnvsToCapture {
		err := validateCaptureConf(conf)
		if err != nil {
			return wrapAsScenarioValidationError(err)
		}
	}

	// check if referred envs in current step has already been defined or not
	if err := checkEnvsValidInStep(si, definedEnvs); err != nil {
		return wrapAsScenarioValidationError(err)
	}

	return nil
}

func wrapAsScenarioValidationError(err error) ScenarioValidationError {
	return ScenarioValidationError{
		msg:        fmt.Sprintf("ScenarioValidationError %v", err),
		wrappedErr: err,
	}
}

func validateCaptureConf(conf EnvCaptureConf) error {
	if !(conf.From == Header || conf.From == Body || conf.From == Cookie) {
		return CaptureConfigError{
			msg: fmt.Sprintf("invalid \"from\" type in capture env : %s", conf.From),
		}
	}

	if conf.From == Header && conf.Key == nil {
		return CaptureConfigError{
			msg: fmt.Sprintf("%s, header key must be specified", conf.Name),
		}
	}

	if conf.From == Body && conf.JsonPath == nil && conf.RegExp == nil && conf.Xpath == nil && conf.XpathHtml == nil {
		return CaptureConfigError{
			msg: fmt.Sprintf("%s, one of json_path, regexp, xpath or xpath_html key must be specified when extracting from body", conf.Name),
		}
	}

	return nil
}

func ParseTLS(certFile, keyFile string) (tls.Certificate, *x509.CertPool, error) {
	if certFile == "" || keyFile == "" {
		return tls.Certificate{}, nil, nil
	}

	// Read the key pair to create certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	// Create a CA certificate pool and add cert.pem to it
	caCert, err := ioutil.ReadFile(certFile)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)

	return cert, pool, nil
}

func IsTargetValid(url string) error {
	if !envVarRegexp.MatchString(url) && !validator.IsURL(strings.ReplaceAll(url, " ", "_")) {
		return fmt.Errorf("target is not valid: %s", url)
	}
	return nil
}
