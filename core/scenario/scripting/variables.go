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
		"guid":         vi.faker.RandomGuid,
		"timestamp":    vi.faker.CurrentTimestamp,
		"isoTimestamp": vi.faker.CurrentISOTimestamp,
		"randomUUID":   vi.faker.RandomUUID,

		//Text, numbers, and colors
		"randomAlphaNumeric": vi.faker.RandomAlphanumeric,
		"randomBoolean":      vi.faker.RandomBoolean,
		"randomInt":          vi.faker.RandomInt,
		"randomColor":        vi.faker.RandomSafeColorName,
		"randomHexColor":     vi.faker.RandomSafeColorHex,
		"randomAbbreviation": vi.faker.RandomAbbreviation,

		// Internet and IP addresses
		"randomIP":         vi.faker.RandomIP,
		"randomIPV6":       vi.faker.RandomIpv6,
		"randomMACAddress": vi.faker.RandomMACAddress,
		"randomPassword":   vi.faker.RandomPassword,
		"randomLocale":     vi.faker.RandomLocale,
		"randomUserAgent":  vi.faker.RandomUserAgent,
		"randomProtocol":   vi.faker.RandomProtocol,
		"randomSemver":     vi.faker.RandomSemver,

		// Names
		"randomFirstName":  vi.faker.RandomPersonFirstName,
		"randomLastName":   vi.faker.RandomPersonLastName,
		"randomFullName":   vi.faker.RandomPersonFullName,
		"randomNamePrefix": vi.faker.RandomPersonNamePrefix,
		"randomNameSuffix": vi.faker.RandomPersonNameSuffix,

		// Profession
		"randomJobArea":       vi.faker.RandomJobArea,
		"randomJobDescriptor": vi.faker.RandomJobDescriptor,
		"randomJobTitle":      vi.faker.RandomJobTitle,
		"randomJobType":       vi.faker.RandomJobType,

		// Phone, address, and location
		"randomPhoneNumber":    vi.faker.RandomPhoneNumber,
		"randomPhoneNumberExt": vi.faker.RandomPhoneNumberExt,
		"randomCity":           vi.faker.RandomAddressCity,
		"randomStreetName":     vi.faker.RandomAddresStreetName,
		"randomStreetAddress":  vi.faker.RandomAddressStreetAddress,
		"randomCountry":        vi.faker.RandomAddressCountry,
		"randomCountryCode":    vi.faker.RandomCountryCode,
		"randomLatitude":       vi.faker.RandomAddressLatitude,
		"randomLongitude":      vi.faker.RandomAddressLongitude,

		// Images
		"randomAvatarImage":    vi.faker.RandomAvatarImage,
		"randomImageUrl":       vi.faker.RandomImageURL,
		"randomAbstractImage":  vi.faker.RandomAbstractImage,
		"randomAnimalsImage":   vi.faker.RandomAnimalsImage,
		"randomBusinessImage":  vi.faker.RandomBusinessImage,
		"randomCatsImage":      vi.faker.RandomCatsImage,
		"randomCityImage":      vi.faker.RandomCityImage,
		"randomFoodImage":      vi.faker.RandomFoodImage,
		"randomNightlifeImage": vi.faker.RandomNightlifeImage,
		"randomFashionImage":   vi.faker.RandomFashionImage,
		"randomPeopleImage":    vi.faker.RandomPeopleImage,
		"randomNatureImage":    vi.faker.RandomNatureImage,
		"randomSportsImage":    vi.faker.RandomSportsImage,
		"randomTransportImage": vi.faker.RandomTransportImage,
		"randomImageDataUri":   vi.faker.RandomDataImageUri,

		// Finance
		"randomBankAccount":     vi.faker.RandomBankAccount,
		"randomBankAccountName": vi.faker.RandomBankAccountName,
		"randomCreditCardMask":  vi.faker.RandomCreditCardMask,
		"randomBankAccountBic":  vi.faker.RandomBankAccountBic,
		"randomBankAccountIban": vi.faker.RandomBankAccountIban,
		"randomTransactionType": vi.faker.RandomTransactionType,
		"randomCurrencyCode":    vi.faker.RandomCurrencyCode,
		"randomCurrencyName":    vi.faker.RandomCurrencyName,
		"randomCurrencySymbol":  vi.faker.RandomCurrencySymbol,
		"randomBitcoin":         vi.faker.RandomBitcoin,

		// Business
		"randomCompanyName":   vi.faker.RandomCompanyName,
		"randomCompanySuffix": vi.faker.RandomCompanySuffix,
		"randomBs":            vi.faker.RandomBs,
		"randomBsAdjective":   vi.faker.RandomBsAdjective,
		"randomBsBuzz":        vi.faker.RandomBsBuzzWord,
		"randomBsNoun":        vi.faker.RandomBsNoun,

		// Catchphrases
		"randomCatchPhrase":           vi.faker.RandomCatchPhrase,
		"randomCatchPhraseAdjective":  vi.faker.RandomCatchPhraseAdjective,
		"randomCatchPhraseDescriptor": vi.faker.RandomCatchPhraseDescriptor,
		"randomCatchPhraseNoun":       vi.faker.RandomCatchPhraseNoun,

		// Databases
		"randomDatabaseColumn":    vi.faker.RandomDatabaseColumn,
		"randomDatabaseType":      vi.faker.RandomDatabaseType,
		"randomDatabaseCollation": vi.faker.RandomDatabaseCollation,
		"randomDatabaseEngine":    vi.faker.RandomDatabaseEngine,

		// Dates
		"randomDateFuture": vi.faker.RandomDateFuture,
		"randomDatePast":   vi.faker.RandomDatePast,
		"randomDateRecent": vi.faker.RandomDateRecent,
		"randomWeekday":    vi.faker.RandomWeekday,
		"randomMonth":      vi.faker.RandomMonth,

		// Domains, emails, and usernames
		"randomDomainName":   vi.faker.RandomDomainName,
		"randomDomainSuffix": vi.faker.RandomDomainSuffix,
		"randomDomainWord":   vi.faker.RandomDomainWord,
		"randomEmail":        vi.faker.RandomEmail,
		"randomExampleEmail": vi.faker.RandomExampleEmail,
		"randomUserName":     vi.faker.RandomUsername,
		"randomUrl":          vi.faker.RandomUrl,

		// Files and directories
		"randomFileName":       vi.faker.RandomFileName,
		"randomFileType":       vi.faker.RandomFileType,
		"randomFileExt":        vi.faker.RandomFileExtension,
		"randomCommonFileName": vi.faker.RandomCommonFileName,
		"randomCommonFileType": vi.faker.RandomCommonFileType,
		"randomCommonFileExt":  vi.faker.RandomCommonFileExtension,
		"randomFilePath":       vi.faker.RandomFilePath,
		"randomDirectoryPath":  vi.faker.RandomDirectoryPath,
		"randomMimeType":       vi.faker.RandomMimeType,

		// Stores
		"randomPrice":            vi.faker.RandomPrice,
		"randomProduct":          vi.faker.RandomProduct,
		"randomProductAdjective": vi.faker.RandomProductAdjective,
		"randomProductMaterial":  vi.faker.RandomProductMaterial,
		"randomProductName":      vi.faker.RandomProductName,
		"randomDepartment":       vi.faker.RandomDepartment,

		// Grammar
		"randomNoun":      vi.faker.RandomNoun,
		"randomVerb":      vi.faker.RandomVerb,
		"randomIngverb":   vi.faker.RandomIngVerb,
		"randomAdjective": vi.faker.RandomAdjective,
		"randomWord":      vi.faker.RandomWord,
		"randomWords":     vi.faker.RandomWords,
		"randomPhrase":    vi.faker.RandomPhrase,

		// Lorem ipsum
		"randomLoremWord":       vi.faker.RandomLoremWord,
		"randomLoremWords":      vi.faker.RandomLoremWords,
		"randomLoremSentence":   vi.faker.RandomLoremSentence,
		"randomLoremSentences":  vi.faker.RandomLoremSentences,
		"randomLoremParagraph":  vi.faker.RandomLoremParagraph,
		"randomLoremParagraphs": vi.faker.RandomLoremParagraphs,
		"randomLoremText":       vi.faker.RandomLoremText,
		"randomLoremSlug":       vi.faker.RandomLoremSlug,
		"randomLoremLines":      vi.faker.RandomLoremLines,

		/*
		 * Spesific to us.
		 */
		"randomFloat":  vi.faker.RandomFloat,
		"randomString": vi.faker.RandomString,
	}

}

func (vi *VariableInjector) Inject(text string) (string, error) {
	return vi.fakeDataInjector(text)
}
func (vi *VariableInjector) fakeDataInjector(text string) (string, error) {
	var err error
	template, err := fasttemplate.NewTemplate(text, "{{_", "}}")
	if err != nil {
		return "", err
	}

	parsed := template.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
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
		case bool:
			p = strconv.FormatBool(res.(bool))
		default:
			p = res.(string)
		}
		return w.Write([]byte(p))

	})
	return parsed, err
}
