package scripting

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jaswdr/faker"

	"go.ddosify.com/ddosify/core/util"
)

type VariableInjector struct {
	faker       faker.Faker
	customFaker util.CustomFaker
	fakerMap    map[string]interface{}
}

func (vi *VariableInjector) Init() {
	vi.faker = faker.New()
	vi.customFaker = util.NewCustomFaker()
	vi.fakerMap = template.FuncMap{
		/*
		* Postman equivalents: https://learning.postman.com/docs/writing-scripts/script-references/variables-list
		 */

		// Common
		"_guid":         uuid.New,
		"_timestamp":    randomTimestamp,
		"_isoTimestamp": randomISOTimestamp,
		"_randomUUID":   uuid.New,

		//Text, numbers, and colors
		//"_randomAlphaNumeric": vi.faker.Letter,
		"_randomBoolean":  vi.faker.Bool,
		"_randomInt":      vi.faker.Int,
		"_randomColor":    vi.faker.Color().SafeColorName,
		"_randomHexColor": vi.faker.Color().Hex,
		// "_randomAbbreviation": to_be_filled,

		// Internet and IP addresses
		"_randomIP": vi.faker.Internet().Ipv4,
		//"_randomIPV6":       vi.faker.Internet().Ipv6,
		"_randomMACAddress": vi.faker.Internet().MacAddress,
		"_randomPassword":   vi.faker.Internet().Password,
		"_randomLocale":     vi.faker.Language().LanguageAbbr,
		"_randomUserAgent":  vi.faker.UserAgent().UserAgent,
		// "randomProtocol":  to_be_filled,,
		"_randomSemver": vi.faker.App().Version,

		// Names
		"_randomFirstName":  vi.faker.Person().FirstName,
		"_randomLastName":   vi.faker.Person().LastName,
		"_randomFullName":   vi.faker.Person().Name,
		"_randomNamePrefix": vi.faker.Person().Title,
		"_randomNameSuffix": vi.faker.Person().Suffix,

		// Profession
		// "_randomJobArea":       vi.faker.Company().JobTitle,
		// "_randomJobDescriptor": vi.faker.Company().JobTitle,
		"_randomJobTitle": vi.faker.Company().JobTitle,
		// "randomJobType":        vi.faker.Company().JobTitle,

		/*
		* Spesific to us.
		 */
		"_randomFloat":  vi.customFaker.RandomFloat,
		"_randomString": vi.customFaker.RandomString,

		// Functions
		"_intBetween":      vi.faker.IntBetween,
		"_floatBetween":    vi.faker.RandomFloat,
		"_stringMaxLength": vi.faker.RandomStringWithLength,
	}

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

func randomTimestamp() int64 {
	randomTime := rand.Int63n(time.Now().Unix()-94608000) + 94608000
	return time.Unix(randomTime, 0).Unix()
}

func randomISOTimestamp() string {
	randomTime := rand.Int63n(time.Now().Unix()-94608000) + 94608000
	return time.Unix(randomTime, 0).UTC().Format("2006-01-02T15:04:05.000Z")
}

func randomInt() int {
	return faker.New().IntBetween(0, 1000000000)
}

func randomFloat() float64 {
	return faker.New().RandomFloat(1, 50, 100)
}

func randomString() string {
	return faker.New().RandomStringWithLength(10)
}
