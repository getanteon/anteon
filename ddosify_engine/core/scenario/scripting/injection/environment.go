package injection

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"go.ddosify.com/ddosify/core/types/regex"
)

type BodyPiece struct {
	start      int
	end        int // end is not inclusive
	injectable bool
	value      string // []byte // exist only if injectable is true
}

type DdosifyBodyReader struct {
	Body   string // []byte
	Pieces []BodyPiece

	// keeps track of the current read position
	pi int // piece index
	vi int // index in the value of the current piece
}

// no-op close
func (dbr *DdosifyBodyReader) Close() error { return nil }

func (dbr *DdosifyBodyReader) Read(dst []byte) (n int, err error) {
	leftSpaceOnDst := len(dst) // assume dst is empty, so we can write to it from the beginning

	var readUntilPieceIndex int
	var readUntilPieceValueIndex int

	readUntilPieceIndex = dbr.pi
	readUntilPieceValueIndex = dbr.vi

	// find the piece index and value index to read until
	for leftSpaceOnDst > 0 && readUntilPieceIndex < len(dbr.Pieces) {
		var unReadOnCurrentPiece int
		piece := dbr.Pieces[readUntilPieceIndex]
		if piece.injectable { // has injected value
			unReadOnCurrentPiece = len(piece.value[readUntilPieceValueIndex:])
		} else {
			unReadOnCurrentPiece = piece.end - piece.start - readUntilPieceValueIndex
		}

		if unReadOnCurrentPiece > leftSpaceOnDst {
			// will be a partial read
			// set readUntilPieceIndex and readUntilPieceValueIndex
			readUntilPieceValueIndex += leftSpaceOnDst
			leftSpaceOnDst = 0
		} else {
			// will be a full read of the current piece
			// set readUntilPieceIndex and readUntilPieceValueIndex
			leftSpaceOnDst -= unReadOnCurrentPiece
			readUntilPieceValueIndex += unReadOnCurrentPiece

			if leftSpaceOnDst > 0 {
				// there is still space on dst
				readUntilPieceIndex++
				readUntilPieceValueIndex = 0
			}
		}
	}

	// continue reading from pieceIndex and valIndex
	// in first iteration, read from dbr.valIndex till the end of the piece
	// later on read from the beginning of the piece till the end of the piece
	for i := dbr.pi; i <= readUntilPieceIndex; i++ {
		if i == len(dbr.Pieces) {
			dbr.pi = i
			return n, io.EOF
		}
		piece := dbr.Pieces[i]
		if piece.injectable {
			// if dst has enough space to hold the whole piece
			// copy the whole piece from where we left off
			if len(dst[n:]) >= len(piece.value)-dbr.vi {
				copy(dst[n:n+len(piece.value)-dbr.vi], piece.value[dbr.vi:])
				n += len(piece.value) - dbr.vi
			} else {
				// if dst does not have enough space to hold the whole piece
				// copy as much as we can and return
				leftSpaceOnDst := len(dst[n:])
				copy(dst[n:], piece.value[dbr.vi:dbr.vi+leftSpaceOnDst])
				n += leftSpaceOnDst
				dbr.pi = i
				dbr.vi = dbr.vi + leftSpaceOnDst
				return n, nil
			}
		} else {
			// if dst has enough space to hold the whole piece
			// copy the whole piece from where we left off
			if len(dst[n:]) >= piece.end-piece.start-dbr.vi {
				copy(dst[n:n+piece.end-piece.start-dbr.vi], dbr.Body[piece.start+dbr.vi:piece.end])
				n += piece.end - piece.start - dbr.vi
			} else {
				// if dst does not have enough space to hold the whole piece
				// copy as much as we can and return
				leftSpaceOnDst := len(dst[n:])
				copy(dst[n:], dbr.Body[piece.start+dbr.vi:piece.start+dbr.vi+leftSpaceOnDst])
				n += leftSpaceOnDst
				dbr.pi = i
				dbr.vi = dbr.vi + leftSpaceOnDst
				return n, nil
			}
		}
		// jump to the next piece
		dbr.vi = 0
	}

	// check if we have reached the end of the body
	if readUntilPieceIndex == len(dbr.Pieces)-1 {
		piece := dbr.Pieces[readUntilPieceIndex]
		if piece.injectable && readUntilPieceValueIndex == len(piece.value) {
			dbr.pi = readUntilPieceIndex + 1 // consumed the whole body
			return n, io.EOF
		} else if !piece.injectable && readUntilPieceValueIndex == piece.end-piece.start {
			dbr.pi = readUntilPieceIndex + 1 // consumed the whole body
			return n, io.EOF
		}
	}

	// readUntilPieceIndex is not the last piece, we fully filled dst
	// set where we left off
	dbr.pi = readUntilPieceIndex
	dbr.vi = readUntilPieceValueIndex
	return n, nil
}

type EnvironmentInjector struct {
	r   *regexp.Regexp
	jr  *regexp.Regexp
	dr  *regexp.Regexp
	jdr *regexp.Regexp
	mu  sync.Mutex
}

func (ei *EnvironmentInjector) Init() {
	ei.r = regexp.MustCompile(regex.EnvironmentVariableRegex)
	ei.jr = regexp.MustCompile(regex.JsonEnvironmentVarRegex)
	ei.dr = regexp.MustCompile(regex.DynamicVariableRegex)
	ei.jdr = regexp.MustCompile(regex.JsonDynamicVariableRegex)
	rand.Seed(time.Now().UnixNano())
}

func truncateTag(tag string, rx string) string {
	if strings.EqualFold(rx, regex.EnvironmentVariableRegex) {
		return tag[2 : len(tag)-2] // {{...}}
	} else if strings.EqualFold(rx, regex.JsonEnvironmentVarRegex) {
		return tag[3 : len(tag)-3] // "{{...}}"
	} else if strings.EqualFold(rx, regex.DynamicVariableRegex) {
		return tag[3 : len(tag)-2] // {{_...}}
	} else if strings.EqualFold(rx, regex.JsonDynamicVariableRegex) {
		return tag[4 : len(tag)-3] //"{{_...}}"
	}
	return ""
}

func (ei *EnvironmentInjector) InjectEnv(text string, envs map[string]interface{}) (string, error) {
	errors := []error{}

	injectStrFunc := getInjectStrFunc(regex.EnvironmentVariableRegex, ei, envs, &errors)
	injectToJsonByteFunc := getInjectJsonFunc(regex.JsonEnvironmentVarRegex, ei, envs, &errors)

	// json injection
	bText := StringToBytes(text)
	if json.Valid(bText) {
		replacedBytes := ei.jr.ReplaceAllFunc(bText, injectToJsonByteFunc)
		if len(errors) == 0 {
			text = string(replacedBytes)
		} else {
			return "", unifyErrors(errors)
		}
	}

	// string injection
	replaced := ei.r.ReplaceAllStringFunc(text, injectStrFunc)
	if len(errors) == 0 {
		return replaced, nil
	}

	return replaced, unifyErrors(errors)

}

func (ei *EnvironmentInjector) getEnv(envs map[string]interface{}, key string) (interface{}, error) {
	var err error
	var val interface{}

	pickRand := strings.HasPrefix(key, "rand(") && strings.HasSuffix(key, ")")
	if pickRand {
		key = key[5 : len(key)-1]
	}

	var exists bool
	val, exists = envs[key]

	isOsEnv := strings.HasPrefix(key, "$")

	if isOsEnv {
		varName := key[1:]
		val, exists = os.LookupEnv(varName)
	}

	if !exists {
		err = fmt.Errorf("env not found")
	}

	if pickRand {
		switch v := val.(type) {
		case []interface{}:
			val = v[rand.Intn(len(v))]
		case []string:
			val = v[rand.Intn(len(v))]
		case []bool:
			val = v[rand.Intn(len(v))]
		case []int:
			val = v[rand.Intn(len(v))]
		case []float64:
			val = v[rand.Intn(len(v))]
		default:
			err = fmt.Errorf("can not perform rand() operation on non-array value")
		}
	}

	return val, err
}

func unifyErrors(errors []error) error {
	sb := strings.Builder{}

	for _, err := range errors {
		sb.WriteString(err.Error())
	}

	return fmt.Errorf("%s", sb.String())
}

func StringToBytes(s string) (b []byte) {
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sliceHeader.Data = stringHeader.Data
	sliceHeader.Len = len(s)
	sliceHeader.Cap = len(s)
	return b
}

func getInjectStrFunc(rx string,
	ei *EnvironmentInjector,
	envs map[string]interface{},
	errors *[]error,
) func(string) string {
	return func(s string) string {
		var truncated string
		var env interface{}
		var err error

		truncated = truncateTag(string(s), rx)

		if rx == regex.EnvironmentVariableRegex {
			env, err = ei.getEnv(envs, truncated)
		} else if rx == regex.DynamicVariableRegex {
			env, err = ei.getFakeData(truncated)
		} else {
			// this should never happen
			panic("invalid regex")
		}

		if err == nil {
			switch env.(type) {
			case string:
				return env.(string)
			case []byte:
				return string(env.([]byte))
			case int64:
				return fmt.Sprintf("%d", env)
			case int:
				return fmt.Sprintf("%d", env)
			case float64:
				return fmt.Sprintf("%g", env) // %g it is the smallest number of digits necessary to identify the value uniquely
			case bool:
				return fmt.Sprintf("%t", env)
			default:
				return fmt.Sprint(env)
			}
		}
		*errors = append(*errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps", truncated))
		return s
	}
}

func getInjectJsonFunc(rx string,
	ei *EnvironmentInjector,
	envs map[string]interface{},
	errors *[]error,
) func(s []byte) []byte {
	return func(s []byte) []byte {
		var truncated string
		var env interface{}
		var err error

		truncated = truncateTag(string(s), rx)
		if rx == regex.JsonDynamicVariableRegex {
			env, err = ei.getFakeData(truncated)
		} else if rx == regex.JsonEnvironmentVarRegex {
			env, err = ei.getEnv(envs, truncated)
		} else {
			// this should never happen
			panic("invalid regex")
		}

		if err == nil {
			mEnv, err := json.Marshal(env)
			if err == nil {
				return mEnv
			}
		}
		*errors = append(*errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps", truncated))
		return s
	}
}

type EnvMatch struct {
	regex string // matched regex
	found []int  // indexes of match
}
type EnvMatchSlice []EnvMatch

func (a EnvMatchSlice) Len() int           { return len(a) }
func (a EnvMatchSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EnvMatchSlice) Less(i, j int) bool { return a[i].found[0] < a[j].found[0] }

func (ei *EnvironmentInjector) GenerateBodyPieces(body string, envs map[string]interface{}) []BodyPiece {
	// generate body pieces
	pieces := make([]BodyPiece, 0)
	matches := EnvMatchSlice{}

	bText := StringToBytes(body)
	if json.Valid(bText) {
		jsonEnvMatches := ei.jr.FindAllStringIndex(body, -1)
		for _, match := range jsonEnvMatches {
			matches = append(matches, EnvMatch{
				regex: regex.JsonEnvironmentVarRegex,
				found: match,
			})
		}

		envsInJsonStringMatches := ei.r.FindAllStringIndex(body, -1)
		for _, match := range envsInJsonStringMatches {
			// exclude ones that are already matched as json envs
			alreadyMatched := false
			for _, jsonMatch := range jsonEnvMatches {
				if match[0] >= jsonMatch[0] && match[1] <= jsonMatch[1] {
					alreadyMatched = true
					break
				}
			}

			if alreadyMatched {
				continue
			}

			matches = append(matches, EnvMatch{
				regex: regex.EnvironmentVariableRegex,
				found: match,
			})
		}

		jsonDynamicMatches := ei.jdr.FindAllStringIndex(body, -1)
		for _, match := range jsonDynamicMatches {
			matches = append(matches, EnvMatch{
				regex: regex.JsonDynamicVariableRegex,
				found: match,
			})
		}

		dynamicEnvsInJsonStringMatches := ei.dr.FindAllStringIndex(body, -1)
		for _, match := range dynamicEnvsInJsonStringMatches {
			// exclude ones that are already matched as json dynamic envs
			alreadyMatched := false
			for _, jsonMatch := range jsonDynamicMatches {
				if match[0] >= jsonMatch[0] && match[1] <= jsonMatch[1] {
					alreadyMatched = true
					break
				}
			}

			if alreadyMatched {
				continue
			}

			matches = append(matches, EnvMatch{
				regex: regex.DynamicVariableRegex,
				found: match,
			})
		}
	} else {
		// not json
		envMatches := ei.r.FindAllStringIndex(body, -1)
		for _, match := range envMatches {
			matches = append(matches, EnvMatch{
				regex: regex.EnvironmentVariableRegex,
				found: match,
			})
		}

		dynamicMathces := ei.dr.FindAllStringIndex(body, -1)
		for _, match := range dynamicMathces {
			matches = append(matches, EnvMatch{
				regex: regex.DynamicVariableRegex,
				found: match,
			})
		}
	}

	sort.Sort(matches) // by start index

	errors := make([]error, 0)
	off := 0

	for _, match := range matches {
		r := match.regex
		start := match.found[0]
		end := match.found[1]

		if start > off {
			pieces = append(pieces, BodyPiece{
				start:      off,
				end:        start,
				injectable: false,
				// value:      body[off:match[0]], // no need to put values in here
			})
		}

		f := getInjectStrFunc(regex.EnvironmentVariableRegex, ei, envs, &errors)
		fd := getInjectStrFunc(regex.DynamicVariableRegex, ei, nil, &errors)

		jf := getInjectJsonFunc(regex.JsonEnvironmentVarRegex, ei, envs, &errors)
		jfd := getInjectJsonFunc(regex.JsonDynamicVariableRegex, ei, nil, &errors)

		getValue := func(s string, r string) string {
			if r == regex.JsonEnvironmentVarRegex {
				return string(jf(StringToBytes(s)))
			} else if r == regex.JsonDynamicVariableRegex {
				return string(jfd(StringToBytes(s)))
			} else if r == regex.EnvironmentVariableRegex {
				return f(s)
			} else if r == regex.DynamicVariableRegex {
				return fd(s)
			}
			return s // this should never happen
		}

		val := getValue(body[start:end], r)

		pieces = append(pieces, BodyPiece{
			start:      start,
			end:        end,
			injectable: true,
			value:      val, // only put values in injected pieces
		})

		off = end
	}

	if off < len(body) {
		pieces = append(pieces, BodyPiece{
			start:      off,
			end:        len(body),
			injectable: false,
			// value:      body[off:],
		})
	}

	return pieces
}

func GetContentLength(pieces []BodyPiece) int {
	var contentLength int
	for _, piece := range pieces {
		if piece.injectable {
			contentLength += len(piece.value)
		} else {
			contentLength += piece.end - piece.start
		}
	}
	return contentLength
}
