package scripting

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jaswdr/faker"
)

type VariableInjector struct {
	faker    faker.Faker
	fakerMap map[string]interface{}
}

func New() *VariableInjector {
	vi := &VariableInjector{}
	vi.faker = faker.New()
	vi.fakerMap = template.FuncMap{
		/*
		* Postman equivalents.
		 */
		"_guid":         uuid.New().String,
		"_randomUUID":   uuid.New().String,
		"_timestamp":    randomTimestamp().Unix,
		"_isoTimestamp": randomISOTimestamp,
		// "_randomAlphaNumeric":  to_be_filled,
		"_randomBoolean":  vi.faker.Bool,
		"_randomInt":      randomInt,
		"_randomColor":    vi.faker.Color().ColorName,
		"_randomHexColor": vi.faker.Color().Hex,
		// "_randomAbbreviation": to_be_filled,
		"_randomIP":         vi.faker.Internet().Ipv4,
		"_randomIPV6":       vi.faker.Internet().Ipv6,
		"_randomMACAddress": vi.faker.Internet().MacAddress,
		"_randomPassword":   vi.faker.Internet().Password,
		"_randomLocale":     vi.faker.Language().LanguageAbbr,
		"_randomUserAgent":  vi.faker.UserAgent().UserAgent,
		// "randomProtocol":  to_be_filled,,
		"_randomSemver":     vi.faker.App().Version,
		"_randomFirstName":  vi.faker.Person().FirstName,
		"_randomLastName":   vi.faker.Person().LastName,
		"_randomFullName":   vi.faker.Person().Name,
		"_randomNamePrefix": vi.faker.Person().Title,
		"_randomNameSuffix": vi.faker.Person().Suffix,

		/*
		* Spesific to us.
		 */
		"_randomFloat":  randomFloat,
		"_randomString": randomString,

		// Functions
		"_intBetween":         vi.faker.IntBetween,
		"_floatBetween":       vi.faker.RandomFloat,
		"_stringWithLength":   vi.faker.RandomStringWithLength,
		"_alphaNumWithLength": alphaNumWithLength,
	}
	return vi
}

func (vi *VariableInjector) Inject(text string) string {
	return vi.fakeDataInjector(text)
}

func (vi *VariableInjector) fakeDataInjector(text string) string {
	parsed, err := template.New("").Funcs(vi.fakerMap).Parse(text)
	if err != nil {
		fmt.Println("ERRR", err)
		return text
	}

	buf := &bytes.Buffer{}
	_ = parsed.Execute(buf, nil)
	return buf.String()
}

func randomTimestamp() time.Time {
	randomTime := rand.Int63n(time.Now().Unix()-94608000) + 94608000
	return time.Unix(randomTime, 0)
}

func randomISOTimestamp() string {
	randomTime := rand.Int63n(time.Now().Unix()-94608000) + 94608000
	return time.Unix(randomTime, 0).UTC().Format("2006-01-02T15:04:05.000Z")
}

func randomInt() int {
	return faker.New().IntBetween(0, 100)
}

func randomFloat() float64 {
	return faker.New().RandomFloat(1, 50, 100)
}

func randomString() string {
	return faker.New().RandomStringWithLength(10)
}

func alphaNumWithLength(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
}
