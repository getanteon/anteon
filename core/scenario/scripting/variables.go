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
		"_randomAlphaNumeric": vi.customFaker.RandomAlphanumeric,
		"_randomBoolean":      vi.faker.Bool,
		"_randomInt":          vi.faker.Int,
		"_randomColor":        vi.faker.Color().SafeColorName,
		"_randomHexColor":     vi.faker.Color().Hex,
		"_randomAbbreviation": vi.customFaker.RandomAbbreviation,

		// Internet and IP addresses
		"_randomIP":         vi.faker.Internet().Ipv4,
		"_randomIPV6":       vi.customFaker.Ipv6,
		"_randomMACAddress": vi.faker.Internet().MacAddress,
		"_randomPassword":   vi.faker.Internet().Password,
		"_randomLocale":     vi.faker.Language().LanguageAbbr,
		"_randomUserAgent":  vi.faker.UserAgent().UserAgent,
		"_randomProtocol":   vi.customFaker.RandomProtocol,
		"_randomSemver":     vi.customFaker.RandomSemver,

		// Names
		"_randomFirstName":  vi.faker.Person().FirstName,
		"_randomLastName":   vi.faker.Person().LastName,
		"_randomFullName":   vi.faker.Person().Name,
		"_randomNamePrefix": vi.faker.Person().Title,
		"_randomNameSuffix": vi.faker.Person().Suffix,

		// Profession
		"_randomJobArea":       vi.customFaker.RandomJobArea,
		"_randomJobDescriptor": vi.customFaker.RandomJobDescriptor,
		"_randomJobTitle":      vi.customFaker.RandomJobTitle,
		"_randomJobType":       vi.customFaker.RandomJobType,

		// Phone, address, and location
		"_randomPhoneNumber":    vi.customFaker.RandomPhoneNumber,
		"_randomPhoneNumberExt": vi.customFaker.RandomPhoneNumberExt,
		"_randomCity":           vi.faker.Address().City,
		"_randomStreetName":     vi.faker.Address().StreetName,
		"_randomStreetAddress":  vi.faker.Address().StreetAddress,
		"_randomCountry":        vi.faker.Address().Country,
		"_randomCountryCode":    vi.customFaker.RandomCountryCode,
		"_randomLatitude":       vi.faker.Address().Latitude,
		"_randomLongitude":      vi.faker.Address().Longitude,

		// Images
		"_randomAvatarImage":    vi.customFaker.RandomAvatarImage,
		"_randomImageUrl":       vi.customFaker.RandomImageURL,
		"_randomAbstractImage":  vi.customFaker.RandomAbstractImage,
		"_randomAnimalsImage":   vi.customFaker.RandomAnimalsImage,
		"_randomBusinessImage":  vi.customFaker.RandomBusinessImage,
		"_randomCatsImage":      vi.customFaker.RandomCatsImage,
		"_randomCityImage":      vi.customFaker.RandomCityImage,
		"_randomFoodImage":      vi.customFaker.RandomFoodImage,
		"_randomNightlifeImage": vi.customFaker.RandomNightlifeImage,
		"_randomFashionImage":   vi.customFaker.RandomFashionImage,
		"_randomPeopleImage":    vi.customFaker.RandomPeopleImage,
		"_randomNatureImage":    vi.customFaker.RandomNatureImage,
		"_randomTransportImage": vi.customFaker.RandomTransportImage,
		// "_randomImageDataUri":   ,

		// Finance
		"_randomBankAccount":     vi.customFaker.RandomBankAccount,
		"_randomBankAccountName": vi.customFaker.RandomBankAccountName,
		"_randomCreditCardMask":  vi.customFaker.RandomCreaditCardMask,
		// "_randomBankAccountBic":  vi.faker,
		// "_randomBankAccountIban": vi.customFaker.RandomAvatarImage,
		"_randomTransactionType": vi.customFaker.RandomTransactionType,
		"_randomCurrencyCode":    vi.customFaker.RandomCurrencyCode,
		"_randomCurrencyName":    vi.customFaker.RandomCurrencyName,
		"_randomCurrencySymbol":  vi.customFaker.RandomCurrencySymbol,
		"_randomBitcoin":         vi.customFaker.RandomBitcoin,

		// Business
		"_randomCompanyName":   vi.customFaker.RandomCompanyName,
		"_randomCompanySuffix": vi.customFaker.RandomCompanySuffix,
		"_randomBs":            vi.customFaker.RandomBs,
		"_randomBsAdjective":   vi.customFaker.RandomBsAdjective,
		"_randomBsBuzz":        vi.customFaker.RandomBsBuzzVerbs,
		"_randomBsNoun":        vi.customFaker.RandomBsNouns,

		// Catchphrases
		"_randomCatchPhrase":           vi.customFaker.RandomCatchPhrase,
		"_randomCatchPhraseAdjective":  vi.customFaker.RandomCatchPhraseAdjective,
		"_randomCatchPhraseDescriptor": vi.customFaker.RandomCatchPhraseDescriptor,
		"_randomCatchPhraseNoun":       vi.customFaker.RandomCatchPhraseNoun,

		// Databases
		"_randomDatabaseColumn":    vi.customFaker.RandomDatabaseColumn,
		"_randomDatabaseType":      vi.customFaker.RandomDatabaseType,
		"_randomDatabaseCollation": vi.customFaker.RandomDatabaseCollation,
		"_randomDatabaseEngine":    vi.customFaker.RandomDatabaseEngine,

		// Dates
		// "_randomDateFuture": vi.faker.date,
		// "_randomDatePast":   vi.customFaker.RandomDatabaseColumn,
		// "_randomDateRecent": vi.customFaker.RandomDatabaseColumn,
		"_randomWeekday": vi.customFaker.RandomWeekday,
		"_randomMonth":   vi.customFaker.RandomMonth,

		// Domains, emails, and usernames
		"_randomDomainName":   vi.customFaker.RandomDomainName,
		"_randomDomainSuffix": vi.customFaker.RandomDomainSuffix,
		"_randomDomainWord":   vi.customFaker.RandomDomainWord,
		"_randomEmail":        vi.customFaker.RandomEmail,
		"_randomExampleEmail": vi.customFaker.RandomExampleEmail,
		"_randomUserName":     vi.customFaker.RandomUsername,
		"_randomUrl":          vi.customFaker.RandomUrl,

		// Files and directories
		"_randomFileName":       vi.customFaker.RandomFileName,
		"_randomFileType":       vi.customFaker.RandomFileType,
		"_randomFileExt":        vi.customFaker.RandomFileExtension,
		"_randomCommonFileName": vi.customFaker.RandomCommonFileName,
		"_randomCommonFileType": vi.customFaker.RandomCommonFileType,
		"_randomCommonFileExt":  vi.customFaker.RandomCommonFileExtension,
		"_randomFilePath":       vi.customFaker.RandomFilePath,
		"_randomDirectoryPath":  vi.customFaker.RandomDirectoryPath,
		"_randomMimeType":       vi.customFaker.RandomMimeTypes,

		// Stores
		"_randomPrice":            vi.customFaker.RandomPrice,
		"_randomProduct":          vi.customFaker.RandomProduct,
		"_randomProductAdjective": vi.customFaker.RandomProductAdjective,
		"_randomProductMaterial":  vi.customFaker.RandomProductMaterial,
		"_randomProductName":      vi.customFaker.RandomProductName,
		"_randomDepartment":       vi.customFaker.RandomDepartment,

		// Grammar
		"_randomNoun":      vi.customFaker.RandomNoun,
		"_randomVerb":      vi.customFaker.RandomVerb,
		"_randomIngverb":   vi.customFaker.RandomIngVerb,
		"_randomAdjective": vi.customFaker.RandomAdjective,
		"_randomWord":      vi.customFaker.RandomWord,
		"_randomWords":     vi.customFaker.RandomWords,
		"_randomPhrase":    vi.customFaker.RandomLoremSentences,

		// Lorem ipsum
		"_randomLoremWord":       vi.customFaker.RandomLoremWord,
		"_randomLoremWords":      vi.customFaker.RandomLoremWords,
		"_randomLoremSentence":   vi.customFaker.RandomLoremSentence,
		"_randomLoremSentences":  vi.customFaker.RandomLoremSentences,
		"_randomLoremParagraph":  vi.customFaker.RandomLoremParagraph,
		"_randomLoremParagraphs": vi.customFaker.RandomLoremParagraphs,
		"_randomLoremText":       vi.customFaker.RandomLoremSentences,
		"_randomLoremSlug":       vi.customFaker.RandomLoremSlug,
		// "_randomLoremLines":      vi.customFaker.RandomLoremLines,

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
