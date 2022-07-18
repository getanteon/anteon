package scripting

import (
	"fmt"
	"io"
	"reflect"
	"strconv"

	"github.com/ddosify/go-faker/faker"
	"github.com/google/uuid"
	"github.com/valyala/fasttemplate"
)

type VariableInjector struct {
	faker    faker.Faker
	fakerMap map[string]interface{}
}

func (vi *VariableInjector) Init() {
	vi.faker = faker.NewFaker()
	vi.fakerMap = map[string]interface{}{
		/*
		* Postman equivalents: https://learning.postman.com/docs/writing-scripts/script-references/variables-list
		 */

		// Common
		"_guid":         vi.faker.RandomGuid,
		"_timestamp":    vi.faker.CurrentTimestamp,
		"_isoTimestamp": vi.faker.CurrentISOTimestamp,
		"_randomUUID":   vi.faker.RandomUUID,

		//Text, numbers, and colors
		"_randomAlphaNumeric": vi.faker.RandomAlphanumeric,
		"_randomBoolean":      vi.faker.RandomBoolean,
		"_randomInt":          vi.faker.RandomInt,
		"_randomColor":        vi.faker.RandomSafeColorName,
		"_randomHexColor":     vi.faker.RandomSafeColorHex,
		"_randomAbbreviation": vi.faker.RandomAbbreviation,

		// Internet and IP addresses
		"_randomIP":         vi.faker.RandomIP,
		"_randomIPV6":       vi.faker.RandomIpv6,
		"_randomMACAddress": vi.faker.RandomMACAddress,
		"_randomPassword":   vi.faker.RandomPassword,
		"_randomLocale":     vi.faker.RandomLocale,
		"_randomUserAgent":  vi.faker.RandomUserAgent,
		"_randomProtocol":   vi.faker.RandomProtocol,
		"_randomSemver":     vi.faker.RandomSemver,

		// Names
		"_randomFirstName":  vi.faker.RandomPersonFirstName,
		"_randomLastName":   vi.faker.RandomPersonLastName,
		"_randomFullName":   vi.faker.RandomPersonFullName,
		"_randomNamePrefix": vi.faker.RandomPersonNamePrefix,
		"_randomNameSuffix": vi.faker.RandomPersonNameSuffix,

		// Profession
		"_randomJobArea":       vi.faker.RandomJobArea,
		"_randomJobDescriptor": vi.faker.RandomJobDescriptor,
		"_randomJobTitle":      vi.faker.RandomJobTitle,
		"_randomJobType":       vi.faker.RandomJobType,

		// Phone, address, and location
		"_randomPhoneNumber":    vi.faker.RandomPhoneNumber,
		"_randomPhoneNumberExt": vi.faker.RandomPhoneNumberExt,
		"_randomCity":           vi.faker.RandomAddressCity,
		"_randomStreetName":     vi.faker.RandomAddresStreetName,
		"_randomStreetAddress":  vi.faker.RandomAddressStreetAddress,
		"_randomCountry":        vi.faker.RandomAddressCountry,
		"_randomCountryCode":    vi.faker.RandomCountryCode,
		"_randomLatitude":       vi.faker.RandomAddressLatitude,
		"_randomLongitude":      vi.faker.RandomAddressLongitude,

		// Images
		"_randomAvatarImage":    vi.faker.RandomAvatarImage,
		"_randomImageUrl":       vi.faker.RandomImageURL,
		"_randomAbstractImage":  vi.faker.RandomAbstractImage,
		"_randomAnimalsImage":   vi.faker.RandomAnimalsImage,
		"_randomBusinessImage":  vi.faker.RandomBusinessImage,
		"_randomCatsImage":      vi.faker.RandomCatsImage,
		"_randomCityImage":      vi.faker.RandomCityImage,
		"_randomFoodImage":      vi.faker.RandomFoodImage,
		"_randomNightlifeImage": vi.faker.RandomNightlifeImage,
		"_randomFashionImage":   vi.faker.RandomFashionImage,
		"_randomPeopleImage":    vi.faker.RandomPeopleImage,
		"_randomNatureImage":    vi.faker.RandomNatureImage,
		"_randomSportsImage":    vi.faker.RandomSportsImage,
		"_randomTransportImage": vi.faker.RandomTransportImage,
		"_randomImageDataUri":   vi.faker.RandomDataImageUri,

		// Finance
		"_randomBankAccount":     vi.faker.RandomBankAccount,
		"_randomBankAccountName": vi.faker.RandomBankAccountName,
		"_randomCreditCardMask":  vi.faker.RandomCreditCardMask,
		"_randomBankAccountBic":  vi.faker.RandomBankAccountBic,
		"_randomBankAccountIban": vi.faker.RandomBankAccountIban,
		"_randomTransactionType": vi.faker.RandomTransactionType,
		"_randomCurrencyCode":    vi.faker.RandomCurrencyCode,
		"_randomCurrencyName":    vi.faker.RandomCurrencyName,
		"_randomCurrencySymbol":  vi.faker.RandomCurrencySymbol,
		"_randomBitcoin":         vi.faker.RandomBitcoin,

		// Business
		"_randomCompanyName":   vi.faker.RandomCompanyName,
		"_randomCompanySuffix": vi.faker.RandomCompanySuffix,
		"_randomBs":            vi.faker.RandomBs,
		"_randomBsAdjective":   vi.faker.RandomBsAdjective,
		"_randomBsBuzz":        vi.faker.RandomBsBuzzWord,
		"_randomBsNoun":        vi.faker.RandomBsNoun,

		// Catchphrases
		"_randomCatchPhrase":           vi.faker.RandomCatchPhrase,
		"_randomCatchPhraseAdjective":  vi.faker.RandomCatchPhraseAdjective,
		"_randomCatchPhraseDescriptor": vi.faker.RandomCatchPhraseDescriptor,
		"_randomCatchPhraseNoun":       vi.faker.RandomCatchPhraseNoun,

		// Databases
		"_randomDatabaseColumn":    vi.faker.RandomDatabaseColumn,
		"_randomDatabaseType":      vi.faker.RandomDatabaseType,
		"_randomDatabaseCollation": vi.faker.RandomDatabaseCollation,
		"_randomDatabaseEngine":    vi.faker.RandomDatabaseEngine,

		// Dates
		"_randomDateFuture": vi.faker.RandomDateFuture,
		"_randomDatePast":   vi.faker.RandomDatePast,
		"_randomDateRecent": vi.faker.RandomDateRecent,
		"_randomWeekday":    vi.faker.RandomWeekday,
		"_randomMonth":      vi.faker.RandomMonth,

		// Domains, emails, and usernames
		"_randomDomainName":   vi.faker.RandomDomainName,
		"_randomDomainSuffix": vi.faker.RandomDomainSuffix,
		"_randomDomainWord":   vi.faker.RandomDomainWord,
		"_randomEmail":        vi.faker.RandomEmail,
		"_randomExampleEmail": vi.faker.RandomExampleEmail,
		"_randomUserName":     vi.faker.RandomUsername,
		"_randomUrl":          vi.faker.RandomUrl,

		// Files and directories
		"_randomFileName":       vi.faker.RandomFileName,
		"_randomFileType":       vi.faker.RandomFileType,
		"_randomFileExt":        vi.faker.RandomFileExtension,
		"_randomCommonFileName": vi.faker.RandomCommonFileName,
		"_randomCommonFileType": vi.faker.RandomCommonFileType,
		"_randomCommonFileExt":  vi.faker.RandomCommonFileExtension,
		"_randomFilePath":       vi.faker.RandomFilePath,
		"_randomDirectoryPath":  vi.faker.RandomDirectoryPath,
		"_randomMimeType":       vi.faker.RandomMimeType,

		// Stores
		"_randomPrice":            vi.faker.RandomPrice,
		"_randomProduct":          vi.faker.RandomProduct,
		"_randomProductAdjective": vi.faker.RandomProductAdjective,
		"_randomProductMaterial":  vi.faker.RandomProductMaterial,
		"_randomProductName":      vi.faker.RandomProductName,
		"_randomDepartment":       vi.faker.RandomDepartment,

		// Grammar
		"_randomNoun":      vi.faker.RandomNoun,
		"_randomVerb":      vi.faker.RandomVerb,
		"_randomIngverb":   vi.faker.RandomIngVerb,
		"_randomAdjective": vi.faker.RandomAdjective,
		"_randomWord":      vi.faker.RandomWord,
		"_randomWords":     vi.faker.RandomWords,
		"_randomPhrase":    vi.faker.RandomPhrase,

		// Lorem ipsum
		"_randomLoremWord":       vi.faker.RandomLoremWord,
		"_randomLoremWords":      vi.faker.RandomLoremWords,
		"_randomLoremSentence":   vi.faker.RandomLoremSentence,
		"_randomLoremSentences":  vi.faker.RandomLoremSentences,
		"_randomLoremParagraph":  vi.faker.RandomLoremParagraph,
		"_randomLoremParagraphs": vi.faker.RandomLoremParagraphs,
		"_randomLoremText":       vi.faker.RandomLoremText,
		"_randomLoremSlug":       vi.faker.RandomLoremSlug,
		"_randomLoremLines":      vi.faker.RandomLoremLines,

		/*
		 * Spesific to us.
		 */
		"_randomFloat":  vi.faker.RandomFloat,
		"_randomString": vi.faker.RandomString,
	}

}

func (vi *VariableInjector) Inject(text string) (string, error) {
	return vi.fakeDataInjector(text)
}
func (vi *VariableInjector) fakeDataInjector(text string) (string, error) {
	var err error
	parsed := fasttemplate.New(text, "{{", "}}").ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		if _, ok := vi.fakerMap[tag]; !ok {
			err = fmt.Errorf("%s is not a valid dynamic variable", tag)
			return 0, nil
		}

		res := reflect.ValueOf(vi.fakerMap[tag]).Call(nil)[0].Interface()

		var p string
		switch res.(type) {
		case int:
			p = strconv.Itoa(res.(int))
		case int64:
			p = strconv.FormatInt(res.(int64), 10)
		case float64:
			p = fmt.Sprintf("%f", res.(float64))
		case uuid.UUID:
			p = res.(uuid.UUID).String()
		default:
			p = res.(string)
		}
		return w.Write([]byte(p))

	})
	return parsed, err
}
