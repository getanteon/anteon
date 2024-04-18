package injection

import "github.com/ddosify/go-faker/faker"

var dynamicFakeDataMap map[string]interface{}
var dataFaker faker.Faker

func init() {
	dataFaker = faker.NewFaker()
	dynamicFakeDataMap = map[string]interface{}{
		/*
		* Postman equivalents: https://learning.postman.com/docs/writing-scripts/script-references/variables-list
		 */

		// Common
		"guid":         dataFaker.RandomGuid,
		"timestamp":    dataFaker.CurrentTimestamp,
		"isoTimestamp": dataFaker.CurrentISOTimestamp,
		"randomUUID":   dataFaker.RandomUUID,

		//Text, numbers, and colors
		"randomAlphaNumeric": dataFaker.RandomAlphanumeric,
		"randomBoolean":      dataFaker.RandomBoolean,
		"randomInt":          dataFaker.RandomInt,
		"randomColor":        dataFaker.RandomSafeColorName,
		"randomHexColor":     dataFaker.RandomSafeColorHex,
		"randomAbbreviation": dataFaker.RandomAbbreviation,

		// Internet and IP addresses
		"randomIP":         dataFaker.RandomIP,
		"randomIPV6":       dataFaker.RandomIpv6,
		"randomMACAddress": dataFaker.RandomMACAddress,
		"randomPassword":   dataFaker.RandomPassword,
		"randomLocale":     dataFaker.RandomLocale,
		"randomUserAgent":  dataFaker.RandomUserAgent,
		"randomProtocol":   dataFaker.RandomProtocol,
		"randomSemver":     dataFaker.RandomSemver,

		// Names
		"randomFirstName":  dataFaker.RandomPersonFirstName,
		"randomLastName":   dataFaker.RandomPersonLastName,
		"randomFullName":   dataFaker.RandomPersonFullName,
		"randomNamePrefix": dataFaker.RandomPersonNamePrefix,
		"randomNameSuffix": dataFaker.RandomPersonNameSuffix,

		// Profession
		"randomJobArea":       dataFaker.RandomJobArea,
		"randomJobDescriptor": dataFaker.RandomJobDescriptor,
		"randomJobTitle":      dataFaker.RandomJobTitle,
		"randomJobType":       dataFaker.RandomJobType,

		// Phone, address, and location
		"randomPhoneNumber":    dataFaker.RandomPhoneNumber,
		"randomPhoneNumberExt": dataFaker.RandomPhoneNumberExt,
		"randomCity":           dataFaker.RandomAddressCity,
		"randomStreetName":     dataFaker.RandomAddresStreetName,
		"randomStreetAddress":  dataFaker.RandomAddressStreetAddress,
		"randomCountry":        dataFaker.RandomAddressCountry,
		"randomCountryCode":    dataFaker.RandomCountryCode,
		"randomLatitude":       dataFaker.RandomAddressLatitude,
		"randomLongitude":      dataFaker.RandomAddressLongitude,

		// Images
		"randomAvatarImage":    dataFaker.RandomAvatarImage,
		"randomImageUrl":       dataFaker.RandomImageURL,
		"randomAbstractImage":  dataFaker.RandomAbstractImage,
		"randomAnimalsImage":   dataFaker.RandomAnimalsImage,
		"randomBusinessImage":  dataFaker.RandomBusinessImage,
		"randomCatsImage":      dataFaker.RandomCatsImage,
		"randomCityImage":      dataFaker.RandomCityImage,
		"randomFoodImage":      dataFaker.RandomFoodImage,
		"randomNightlifeImage": dataFaker.RandomNightlifeImage,
		"randomFashionImage":   dataFaker.RandomFashionImage,
		"randomPeopleImage":    dataFaker.RandomPeopleImage,
		"randomNatureImage":    dataFaker.RandomNatureImage,
		"randomSportsImage":    dataFaker.RandomSportsImage,
		"randomTransportImage": dataFaker.RandomTransportImage,
		"randomImageDataUri":   dataFaker.RandomDataImageUri,

		// Finance
		"randomBankAccount":     dataFaker.RandomBankAccount,
		"randomBankAccountName": dataFaker.RandomBankAccountName,
		"randomCreditCardMask":  dataFaker.RandomCreditCardMask,
		"randomBankAccountBic":  dataFaker.RandomBankAccountBic,
		"randomBankAccountIban": dataFaker.RandomBankAccountIban,
		"randomTransactionType": dataFaker.RandomTransactionType,
		"randomCurrencyCode":    dataFaker.RandomCurrencyCode,
		"randomCurrencyName":    dataFaker.RandomCurrencyName,
		"randomCurrencySymbol":  dataFaker.RandomCurrencySymbol,
		"randomBitcoin":         dataFaker.RandomBitcoin,

		// Business
		"randomCompanyName":   dataFaker.RandomCompanyName,
		"randomCompanySuffix": dataFaker.RandomCompanySuffix,
		"randomBs":            dataFaker.RandomBs,
		"randomBsAdjective":   dataFaker.RandomBsAdjective,
		"randomBsBuzz":        dataFaker.RandomBsBuzzWord,
		"randomBsNoun":        dataFaker.RandomBsNoun,

		// Catchphrases
		"randomCatchPhrase":           dataFaker.RandomCatchPhrase,
		"randomCatchPhraseAdjective":  dataFaker.RandomCatchPhraseAdjective,
		"randomCatchPhraseDescriptor": dataFaker.RandomCatchPhraseDescriptor,
		"randomCatchPhraseNoun":       dataFaker.RandomCatchPhraseNoun,

		// Databases
		"randomDatabaseColumn":    dataFaker.RandomDatabaseColumn,
		"randomDatabaseType":      dataFaker.RandomDatabaseType,
		"randomDatabaseCollation": dataFaker.RandomDatabaseCollation,
		"randomDatabaseEngine":    dataFaker.RandomDatabaseEngine,

		// Dates
		"randomDateFuture": dataFaker.RandomDateFuture,
		"randomDatePast":   dataFaker.RandomDatePast,
		"randomDateRecent": dataFaker.RandomDateRecent,
		"randomWeekday":    dataFaker.RandomWeekday,
		"randomMonth":      dataFaker.RandomMonth,

		// Domains, emails, and usernames
		"randomDomainName":   dataFaker.RandomDomainName,
		"randomDomainSuffix": dataFaker.RandomDomainSuffix,
		"randomDomainWord":   dataFaker.RandomDomainWord,
		"randomEmail":        dataFaker.RandomEmail,
		"randomExampleEmail": dataFaker.RandomExampleEmail,
		"randomUserName":     dataFaker.RandomUsername,
		"randomUrl":          dataFaker.RandomUrl,

		// Files and directories
		"randomFileName":       dataFaker.RandomFileName,
		"randomFileType":       dataFaker.RandomFileType,
		"randomFileExt":        dataFaker.RandomFileExtension,
		"randomCommonFileName": dataFaker.RandomCommonFileName,
		"randomCommonFileType": dataFaker.RandomCommonFileType,
		"randomCommonFileExt":  dataFaker.RandomCommonFileExtension,
		"randomFilePath":       dataFaker.RandomFilePath,
		"randomDirectoryPath":  dataFaker.RandomDirectoryPath,
		"randomMimeType":       dataFaker.RandomMimeType,

		// Stores
		"randomPrice":            dataFaker.RandomPrice,
		"randomProduct":          dataFaker.RandomProduct,
		"randomProductAdjective": dataFaker.RandomProductAdjective,
		"randomProductMaterial":  dataFaker.RandomProductMaterial,
		"randomProductName":      dataFaker.RandomProductName,
		"randomDepartment":       dataFaker.RandomDepartment,

		// Grammar
		"randomNoun":      dataFaker.RandomNoun,
		"randomVerb":      dataFaker.RandomVerb,
		"randomIngverb":   dataFaker.RandomIngVerb,
		"randomAdjective": dataFaker.RandomAdjective,
		"randomWord":      dataFaker.RandomWord,
		"randomWords":     dataFaker.RandomWords,
		"randomPhrase":    dataFaker.RandomPhrase,

		// Lorem ipsum
		"randomLoremWord":       dataFaker.RandomLoremWord,
		"randomLoremWords":      dataFaker.RandomLoremWords,
		"randomLoremSentence":   dataFaker.RandomLoremSentence,
		"randomLoremSentences":  dataFaker.RandomLoremSentences,
		"randomLoremParagraph":  dataFaker.RandomLoremParagraph,
		"randomLoremParagraphs": dataFaker.RandomLoremParagraphs,
		"randomLoremText":       dataFaker.RandomLoremText,
		"randomLoremSlug":       dataFaker.RandomLoremSlug,
		"randomLoremLines":      dataFaker.RandomLoremLines,

		/*
		 * Spesific to us.
		 */
		"randomFloat":  dataFaker.RandomFloat,
		"randomString": dataFaker.RandomString,
	}
}
