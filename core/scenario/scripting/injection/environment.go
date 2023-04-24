package injection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"regexp"
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
	pieceIndex int
	valIndex   int
}

// TODO: check bounds
func (dbr *DdosifyBodyReader) Read(dst []byte) (n int, err error) {
	// TODO: check
	leftSpaceOnDst := len(dst) // assume dst is empty, so we can write to it from the beginning

	var readUntilPieceIndex int
	var readUntilPieceValueIndex int

	readUntilPieceIndex = dbr.pieceIndex
	readUntilPieceValueIndex = dbr.valIndex

	for leftSpaceOnDst > 0 {
		var unReadOnCurrentPiece int
		piece := dbr.Pieces[readUntilPieceIndex]
		if piece.injectable { // has injected value
			unReadOnCurrentPiece = len(piece.value[readUntilPieceValueIndex:])
		} else {
			unReadOnCurrentPiece = piece.end - piece.start - dbr.valIndex
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

	// TODO: fails on big first piece

	// continue reading from pieceIndex and valIndex
	// in first iteration, read from dbr.valIndex till the end of the piece
	// later on read from the beginning of the piece till the end of the piece
	for i := dbr.pieceIndex; i <= readUntilPieceIndex; i++ {
		piece := dbr.Pieces[i]
		if piece.injectable {
			// if dst has enough space to hold the whole piece
			// copy the whole piece from where we left off
			if len(dst[n:]) >= len(piece.value)-dbr.valIndex {
				copy(dst[n:n+len(piece.value)-dbr.valIndex], piece.value[dbr.valIndex:])
				n += len(piece.value) - dbr.valIndex
			} else {
				// if dst does not have enough space to hold the whole piece
				// copy as much as we can and return
				leftSpaceOnDst := len(dst[n:])
				copy(dst[n:], piece.value[dbr.valIndex:dbr.valIndex+leftSpaceOnDst])
				n += leftSpaceOnDst
				dbr.pieceIndex = i
				dbr.valIndex = dbr.valIndex + leftSpaceOnDst
				return n, nil
			}

		} else {
			// if dst has enough space to hold the whole piece
			// copy the whole piece from where we left off
			if len(dst[n:]) >= piece.end-piece.start-dbr.valIndex {
				copy(dst[n:n+piece.end-piece.start-dbr.valIndex], dbr.Body[piece.start+dbr.valIndex:piece.end])
				n += piece.end - piece.start - dbr.valIndex
			} else {
				// if dst does not have enough space to hold the whole piece
				// copy as much as we can and return
				leftSpaceOnDst := len(dst[n:])
				copy(dst[n:], dbr.Body[piece.start+dbr.valIndex:piece.start+dbr.valIndex+leftSpaceOnDst])
				n += leftSpaceOnDst
				dbr.pieceIndex = i
				dbr.valIndex = dbr.valIndex + leftSpaceOnDst
				return n, nil
			}
		}
		dbr.valIndex = 0
	}

	// check if EOF
	if readUntilPieceIndex == len(dbr.Pieces)-1 {
		piece := dbr.Pieces[readUntilPieceIndex]
		if piece.injectable && readUntilPieceValueIndex == len(piece.value) {
			return n, io.EOF
		} else if !piece.injectable && readUntilPieceValueIndex == piece.end-piece.start {
			return n, io.EOF
		}
	}

	dbr.pieceIndex = readUntilPieceIndex
	dbr.valIndex = readUntilPieceValueIndex
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

	injectStrFunc := getInjectStrFunc(regex.EnvironmentVariableRegex, ei, envs, errors)
	injectToJsonByteFunc := getInjectJsonFunc(regex.JsonEnvironmentVarRegex, ei, envs, errors)

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

// expects an empty buffer and writes the result to it
func (ei *EnvironmentInjector) InjectEnvIntoBuffer(text string, envs map[string]interface{}, buffer *bytes.Buffer) (*bytes.Buffer, error) {
	// TODO: if did not inject anything, write text to buffer
	errors := []error{}
	if buffer == nil {
		buffer = &bytes.Buffer{}
	}
	injectStrFunc := getInjectStrFunc(regex.EnvironmentVariableRegex, ei, envs, errors)
	injectToJsonByteFunc := getInjectJsonFunc(regex.JsonEnvironmentVarRegex, ei, envs, errors)

	// json injection
	bText := StringToBytes(text)
	if json.Valid(bText) {
		foundMatches := ei.jr.FindAll(bText, -1)
		args := make([]string, 0)
		for _, match := range foundMatches {
			args = append(args, string(match))
			args = append(args, string(injectToJsonByteFunc(match)))
		}

		replacer := strings.NewReplacer(args...)
		_, err := replacer.WriteString(buffer, text)
		if err != nil {
			return nil, err
		}
		if len(errors) == 0 {
			text = buffer.String()
		} else {
			return nil, unifyErrors(errors)
		}
	}

	// continue with string injection
	// string injection
	foundMatches := ei.r.FindAllString(text, -1)
	if len(foundMatches) == 0 {
		return buffer, nil
	} else {
		buffer.Reset()

		args := make([]string, 0)
		for _, match := range foundMatches {
			args = append(args, match)
			args = append(args, injectStrFunc(match))
		}
		replacer := strings.NewReplacer(args...)
		_, err := replacer.WriteString(buffer, text)
		if err != nil {
			return nil, err
		}
	}

	if len(errors) == 0 {
		return buffer, nil
	}

	return nil, unifyErrors(errors)
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
	errors []error,
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
		errors = append(errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps", truncated))
		return s
	}
}

func getInjectJsonFunc(rx string,
	ei *EnvironmentInjector,
	envs map[string]interface{},
	errors []error,
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
		errors = append(errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps", truncated))
		return s
	}
}

func (ei *EnvironmentInjector) GenerateBodyPieces(body string, envs map[string]interface{}) []BodyPiece {
	// generate body pieces
	pieces := make([]BodyPiece, 0)

	// TODO: find matches for all regexes and sort them by start index
	foundMatches := ei.r.FindAllStringIndex(body, -1)

	errors := make([]error, 0)

	off := 0

	for _, match := range foundMatches {
		if match[0] > off {
			pieces = append(pieces, BodyPiece{
				start:      off,
				end:        match[0],
				injectable: false,
				// value:      body[off:match[0]], // no need to put values in here
			})
		}

		f := getInjectStrFunc(regex.EnvironmentVariableRegex, ei, envs, errors)
		val := f(body[match[0]:match[1]])
		pieces = append(pieces, BodyPiece{
			start:      match[0],
			end:        match[1],
			injectable: true,
			value:      val, // only put values in injected pieces
		})

		off = match[1]
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
