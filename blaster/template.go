package blaster

import (
	"bytes"
	"fmt"
	"log"
	"text/template"

	"math/rand"
	"time"

	"eagain.net/go/bech32"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

var builtins = template.FuncMap{
	"rand_int":          randInt,
	"rand_string":       randString,
	"rand_float":        randFloat,
	"rand_datum_hash":   randDatumHash,
	"rand_address":      randAddressPattern,
	"rand_credential":   randCredentialPattern,
	"rand_asset":        randAssetPattern,
	"rand_output_ref":   randOutputReferencePattern,
	"rand_metadata_tag": randMetadataTagPattern,
}

func randInt(from int, to int) interface{} {
	return rand.Intn(to-from) + from
}

func randFloat(from float64, to float64) interface{} {
	return (rand.Float64() * (to - from)) + from
}

func randBlake2b256() [32]byte {
	// Make a buffer with the size of 128 bytes.
	// The generator will fill it with random junk.
	buf := make([]byte, 128)
	_, err := rand.Read(buf)
	if err != nil {
		log.Printf("Error while generating random bytes: %s", err)
		// Fill it with zero bytes.
		buf = make([]byte, 128)
	}
	return blake2b.Sum256(buf)
}

func randBlake2b256Hex() string {
	return fmt.Sprintf("%x", randBlake2b256())
}

func randBlake2b256Bench32() string {
	bs := randBlake2b256()
	str, _ := bech32.Encode("ed25519_pk", bs[:])
	return str
}

func randDatumHash() interface{} {
	return randBlake2b256Hex()
}

func randAddressPattern() interface{} {
	return shuffleSources([]func() string{
		func() string { return "addr1" + randBlake2b256Bench32() },
		func() string { return "stake1" + randBlake2b256Bench32() },
		func() string { return "*" },
	})
}

func randCredentialPattern() interface{} {
	sources := []func() string{
		func() string { return randHexString(64) },
		func() string { return randHexString(56) },
		func() string { return "*" },
	}
	return shuffleSources(sources) + "/" + shuffleSources(sources)
}

func randPolicyIDPattern() string {
	return shuffleSources([]func() string{
		func() string { return randHexString(56) },
		func() string { return "*" },
	})
}

func randAssetNamePattern() string {
	sources := []func() string{func() string { return "*" }}

	for i := 0; i <= 64; i++ {
		sources = append(sources, func() string { return randHexString(i) })
	}

	return shuffleSources(sources)
}

func randAssetPattern() interface{} {
	return randPolicyIDPattern() + "." + randAssetNamePattern()
}

func randOutputIndex() string {
	return shuffleSources([]func() string{
		func() string {
			randDigit := func() int { return rand.Intn(10) }
			return fmt.Sprint(randDigit()*100 + randDigit()*10 + randDigit())
		},
		func() string { return "*" },
	})
}

func randTransactionId() string {
	return randHexString(64)
}

func randOutputReferencePattern() string {
	return randOutputIndex() + "@" + randTransactionId()
}

func randMetadataTagPattern() string {
	return "{" + fmt.Sprint(rand.Intn(9999)) + "}"
}

func shuffleSources(sources []func() string) string {
	n := len(sources)
	if n == 0 {
		return ""
	}
	swap := func(i, j int) { sources[i], sources[j] = sources[j], sources[i] }
	rand.Shuffle(n, swap)
	return sources[0]()
}

func randStringWithAlphabet(alphabet []rune, length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

func randHexString(length int) string {
	return randStringWithAlphabet(
		[]rune("abcdefABCDEF0123456789"), length)
}

func randString(length int) interface{} {
	return randStringWithAlphabet(
		[]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"),
		length)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func parseRenderer(in interface{}) (renderer, error) {
	if in == nil {
		return nil, nil
	}
	switch in := in.(type) {
	case map[string]interface{}:
		out := mapR{}
		for k, v := range in {
			p, err := parseRenderer(v)
			if err != nil {
				return nil, err
			}
			out[k] = p
		}
		return out, nil
	case []interface{}:
		out := sliceR{}
		for _, v := range in {
			p, err := parseRenderer(v)
			if err != nil {
				return nil, err
			}
			out = append(out, p)
		}
		return out, nil
	case string:
		tmpl, err := template.New("t").Funcs(builtins).Parse(in)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return templateR{tmpl}, nil
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, complex64, complex128:
		return nativeR{in}, nil
	default:
		return nil, nil
	}
}

type renderer interface {
	render(data map[string]string) (interface{}, error)
}

type mapR map[string]interface{}

func (m mapR) render(data map[string]string) (interface{}, error) {
	out := map[string]interface{}{}
	for k, v := range m {
		if v, ok := v.(renderer); ok {
			r, err := v.render(data)
			if err != nil {
				return nil, err
			}
			out[k] = r
		} else {
			out[k] = v
		}
	}
	return out, nil
}

type sliceR []interface{}

func (s sliceR) render(data map[string]string) (interface{}, error) {
	out := []interface{}{}
	for _, v := range s {
		if v, ok := v.(renderer); ok {
			r, err := v.render(data)
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		} else {
			out = append(out, v)
		}
	}
	return out, nil
}

type templateR struct {
	*template.Template
}

func (t templateR) render(data map[string]string) (interface{}, error) {
	buf := &bytes.Buffer{}
	err := t.Execute(buf, data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf.String(), nil
}

type native interface{}

type nativeR struct {
	native
}

func (n nativeR) render(data map[string]string) (interface{}, error) {
	return n.native, nil
}
