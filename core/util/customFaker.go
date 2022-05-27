package util

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

const (
	maxUint = ^uint(0)
	minUint = 0
	maxInt  = int(maxUint >> 1)
	minInt  = -maxInt - 1
)

var (
	JobDescriptors = []string{
		"Lead", "Senior", "Direct", "Corporate", "Dynamic", "Future", "Product", "National", "Regional", "District",
		"Central", "Global", "Customer", "Investor", "Dynamic", "International", "Legacy", "Forward", "Internal", "Human",
		"Chief", "Principal"}

	JobAreas = []string{
		"Solutions", "Program", "Brand", "Security", "Research", "Marketing", "Directives", "Implementation",
		"Integration", "Functionality", "Response", "Paradigm", "Tactics", "Identity", "Markets", "Group", "Division",
		"Applications", "Optimization", "Operations", "Infrastructure", "Intranet", "Communications", "Web", "Branding",
		"Quality", "Assurance", "Mobility", "Accounts", "Data", "Creative", "Configuration", "Accountability", "Interactions",
		"Factors", "Usability", "Metrics"}

	JobTypes = []string{
		"Supervisor", "Associate", "Executive", "Liaison", "Officer", "Manager", "Engineer", "Specialist", "Director",
		"Coordinator", "Administrator", "Architect", "Analyst", "Designer", "Planner", "Orchestrator", "Technician",
		"Developer", "Producer", "Consultant", "Assistant", "Facilitator", "Agent", "Representative", "Strategist"}

	Abbreviations = []string{
		"TCP", "HTTP", "SDD", "RAM", "GB", "CSS", "SSL", "AGP", "SQL", "FTP", "PCI", "AI", "ADP", "RSS", "XML", "EXE", "COM",
		"HDD", "THX", "SMTP", "SMS", "USB", "PNG", "SAS", "IB", "SCSI", "JSON", "XSS", "JBOD"}

	CountryCodes = []string{
		"AD", "AE", "AF", "AG", "AI", "AL", "AM", "AO", "AQ", "AR", "AS", "AT", "AU", "AW", "AX", "AZ", "BA", "BB", "BD",
		"BE", "BF", "BG", "BH", "BI", "BJ", "BL", "BM", "BN", "BO", "BQ", "BR", "BS", "BT", "BV", "BW", "BY", "BZ", "CA",
		"CC", "CD", "CF", "CG", "CH", "CI", "CK", "CL", "CM", "CN", "CO", "CR", "CU", "CV", "CW", "CX", "CY", "CZ", "DE",
		"DJ", "DK", "DM", "DO", "DZ", "EC", "EE", "EG", "EH", "ER", "ES", "ET", "FI", "FJ", "FK", "FM", "FO", "FR", "GA",
		"GB", "GD", "GE", "GF", "GG", "GH", "GI", "GL", "GM", "GN", "GP", "GQ", "GR", "GS", "GT", "GU", "GW", "GY", "HK",
		"HM", "HN", "HR", "HT", "HU", "ID", "IE", "IL", "IM", "IN", "IO", "IQ", "IR", "IS", "IT", "JE", "JM", "JO", "JP",
		"KE", "KG", "KH", "KI", "KM", "KN", "KP", "KR", "KW", "KY", "KZ", "LA", "LB", "LC", "LI", "LK", "LR", "LS", "LT",
		"LU", "LV", "LY", "MA", "MC", "MD", "ME", "MF", "MG", "MH", "MK", "ML", "MM", "MN", "MO", "MP", "MQ", "MR", "MS",
		"MT", "MU", "MV", "MW", "MX", "MY", "MZ", "NA", "NC", "NE", "NF", "NG", "NI", "NL", "NO", "NP", "NR", "NU", "NZ",
		"OM", "PA", "PE", "PF", "PG", "PH", "PK", "PL", "PM", "PN", "PR", "PS", "PT", "PW", "PY", "QA", "RE", "RO", "RS",
		"RU", "RW", "SA", "SB", "SC", "SD", "SE", "SG", "SH", "SI", "SJ", "SK", "SL", "SM", "SN", "SO", "SR", "SS", "ST",
		"SV", "SX", "SY", "SZ", "TC", "TD", "TF", "TG", "TH", "TJ", "TK", "TL", "TM", "TN", "TO", "TR", "TT", "TV", "TW",
		"TZ", "UA", "UG", "UM", "US", "UY", "UZ", "VA", "VC", "VE", "VG", "VI", "VN", "VU", "WF", "WS", "YE", "YT", "ZA",
		"ZM", "ZW",
	}

	Protocols = []string{"http", "https"}

	BankTransactionTypes = []string{"deposit", "withdrawal", "payment", "invoice"}

	BankAccounts = []string{
		"Checking", "Savings", "Money Market", "Investment", "Home Loan", "Credit Card", "Auto Loan", "Personal Loan"}

	CurrencyNames = []string{
		"UAE Dirham", "Afghani", "Lek", "Armenian Dram", "Netherlands Antillian Guilder", "Kwanza", "Argentine Peso",
		"Australian Dollar", "Aruban Guilder", "Azerbaijanian Manat", "Convertible Marks", "Barbados Dollar", "Taka",
		"Bulgarian Lev", "Bahraini Dinar", "Burundi Franc", "Bermudian Dollar (customarily known as Bermuda Dollar)",
		"Brunei Dollar", "Boliviano boliviano", "Brazilian Real", "Bahamian Dollar", "Pula", "Belarussian Ruble",
		"Belize Dollar", "Canadian Dollar", "Congolese Franc", "Swiss Franc", "Chilean Peso", "Yuan Renminbi",
		"Colombian Peso", "Costa Rican Colon", "Cuban Peso", "Cuban Peso Convertible", "Cape Verde Escudo", "Czech Koruna",
		"Djibouti Franc", "Danish Krone", "Dominican Peso", "Algerian Dinar", "Kroon", "Egyptian Pound", "Nakfa",
		"Ethiopian Birr", "Euro", "Fiji Dollar", "Falkland Islands Pound", "Pound Sterling", "Lari", "Cedi",
		"Gibraltar Pound", "Dalasi", "Guinea Franc", "Quetzal", "Guyana Dollar", "Hong Kong Dollar", "Lempira",
		"Croatian Kuna", "Gourde", "Forint", "Rupiah", "New Israeli Sheqel", "Bhutanese Ngultrum", "Indian Rupee",
		"Iraqi Dinar", "Iranian Rial", "Iceland Krona", "Jamaican Dollar", "Jordanian Dinar", "Yen", "Kenyan Shilling", "Som",
		"Riel", "Comoro Franc", "North Korean Won", "Won", "Kuwaiti Dinar", "Cayman Islands Dollar", "Tenge", "Kip",
		"Lebanese Pound", "Sri Lanka Rupee", "Liberian Dollar", "Lithuanian Litas", "Latvian Lats", "Libyan Dinar",
		"Moroccan Dirham", "Moldovan Leu", "Malagasy Ariary", "Denar", "Kyat", "Tugrik", "Pataca", "Ouguiya",
		"Mauritius Rupee", "Rufiyaa", "Kwacha", "Mexican Peso", "Malaysian Ringgit", "Tunisian Dinar", "Zambian Kwacha",
		"Metical", "Naira", "Cordoba Oro", "Norwegian Krone", "Nepalese Rupee", "New Zealand Dollar", "Rial Omani",
		"Balboa", "Nuevo Sol", "Kina", "Philippine Peso", "Pakistan Rupee", "Zloty", "Guarani", "Qatari Rial", "New Leu",
		"Serbian Dinar", "Russian Ruble", "Rwanda Franc", "Saudi Riyal", "Solomon Islands Dollar", "Seychelles Rupee",
		"Sudanese Pound", "Swedish Krona", "Singapore Dollar", "Saint Helena Pound", "Leone", "Somali Shilling",
		"Surinam Dollar", "Dobra", "El Salvador Colon", "Syrian Pound", "Lilangeni", "Baht", "Somoni", "Manat",
		"Pa'anga", "Turkish Lira", "Trinidad and Tobago Dollar", "New Taiwan Dollar", "Tanzanian Shilling", "Hryvnia",
		"Uganda Shilling", "US Dollar", "Peso Uruguayo", "Uzbekistan Sum", "Bolivar Fuerte", "Dong", "Vatu", "Tala",
		"CFA Franc BEAC", "Silver", "Gold", "Bond Markets Units European Composite Unit (EURCO)",
		"European Monetary Unit (E.M.U.-6)", "European Unit of Account 9(E.U.A.-9)", "European Unit of Account 17(E.U.A.-17)",
		"East Caribbean Dollar", "SDR", "UIC-Franc", "CFA Franc BCEAO", "Palladium", "CFP Franc", "Platinum",
		"Codes specifically reserved for testing purposes", "Yemeni Rial", "Rand", "Lesotho Loti", "Namibia Dollar",
		"Zimbabwe Dollar",
	}

	CurrencyCodes = []string{
		"AED", "AFN", "ALL", "AMD", "ANG", "AOA", "ARS", "AUD", "AWG", "AZN", "BAM", "BBD", "BDT", "BGN", "BHD", "BIF", "BMD",
		"BND", "BOB", "BRL", "BSD", "BWP", "BYR", "BZD", "CAD", "CDF", "CHF", "CLP", "CNY", "COP", "CRC", "CUP", "CUC", "CVE",
		"CZK", "DJF", "DKK", "DOP", "DZD", "EEK", "EGP", "ERN", "ETB", "EUR", "FJD", "FKP", "GBP", "GEL", "GHS", "GIP", "GMD",
		"GNF", "GTQ", "GYD", "HKD", "HNL", "HRK", "HTG", "HUF", "IDR", "ILS", "BTN", "INR", "IQD", "IRR", "ISK", "JMD", "JOD",
		"JPY", "KES", "KGS", "KHR", "KMF", "KPW", "KRW", "KWD", "KYD", "KZT", "LAK", "LBP", "LKR", "LRD", "LTL", "LVL", "LYD",
		"MAD", "MDL", "MGA", "MKD", "MMK", "MNT", "MOP", "MRO", "MUR", "MVR", "MWK", "MXN", "MYR", "MZN", "NGN", "NIO", "NOK",
		"NPR", "NZD", "OMR", "PAB", "PEN", "PGK", "PHP", "PKR", "PLN", "PYG", "QAR", "RON", "RSD", "RUB", "RWF", "SAR", "SBD",
		"SCR", "SDG", "SEK", "SGD", "SHP", "SLL", "SOS", "SRD", "STN", "SVC", "SYP", "SZL", "THB", "TJS", "TMT", "TND", "TOP",
		"TRY", "TTD", "TWD", "TZS", "UAH", "UGX", "USD", "UYU", "UZS", "VEF", "VND", "VUV", "WST", "XAF", "XAG", "XAU", "XBA",
		"XBB", "XBC", "XBD", "XCD", "XDR", "XFU", "XOF", "XPD", "XPF", "XPT", "XTS", "YER", "ZAR", "LSL", "NAD", "ZMK", "ZWL",
	}

	CurrencySymbols = []string{
		"؋", "Lek", "ƒ", "$", "$", "ƒ", "ман", "KM", "$", "лв", "$", "$", "Bs", "R$", "$", "P", "p.", "BZ$", "$", "CHF", "$",
		"¥", "$", "₡", "₱", "$", "Kč", "kr", "RD$", "£", "€", "$", "£", "£", "£", "Q", "$", "$", "L", "kn", "Ft", "Rp", "₪",
		"Nu", "₹", "﷼", "kr", "J$", "¥", "лв", "៛", "₩", "₩", "$", "лв", "₭", "£", "₨", "$", "Lt", "Ls", "ден", "₮", "₨", "$",
		"RM", "MT", "₦", "C$", "kr", "₨", "$", "﷼", "B/.", "S/.", "Php", "₨", "zł", "Gs", "﷼", "lei", "Дин.", "﷼", "$",
		"₨", "kr", "$", "£", "S", "$", "Db", "₡", "£", "฿", "₺", "TT$", "NT$", "₴", "$", "$U", "лв", "Bs", "₫", "$", "﷼", "R",
		"N$",
	}
)

type CustomFaker struct {
	Generator *rand.Rand
}

func (f CustomFaker) RandomBitcoin() string {
	const letters = "123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

	b := make([]byte, f.Generator.Intn(35-26)+26)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (f CustomFaker) RandomCurrencySymbols() string {
	return CurrencySymbols[f.Generator.Intn(len(CurrencySymbols))]
}

func (f CustomFaker) RandomCurrencyCodes() string {
	return CurrencyCodes[f.Generator.Intn(len(CurrencyCodes))]
}

func (f CustomFaker) RandomCurrencyNames() string {
	return CurrencyNames[f.Generator.Intn(len(CurrencyNames))]
}

func (f CustomFaker) RandomTransactionTypes() string {
	return BankTransactionTypes[f.Generator.Intn(len(BankTransactionTypes))]
}

func (f CustomFaker) RandomCreaditCardMask() string {
	return strconv.Itoa(f.Generator.Intn(9999-1000) + 1000)
}

func (f CustomFaker) RandomBankAccountName() string {
	return BankAccounts[f.Generator.Intn(len(BankAccounts))]
}

func (f CustomFaker) RandomBankAccount() string {
	return strconv.Itoa(f.Generator.Intn(99999999-10000000) + 10000000)
}

func (f CustomFaker) RandomAvatarImage() string {
	return "http://placeimg.com/640/480/people"
}

func (f CustomFaker) RandomImageURL() string {
	return "http://placeimg.com/640/480"
}

func (f CustomFaker) RandomAbstractImage() string {
	return "http://placeimg.com/640/480/abstract"
}

func (f CustomFaker) RandomAnimalsImage() string {
	return "http://placeimg.com/640/480/animals"
}

func (f CustomFaker) RandomBusinessImage() string {
	return "http://placeimg.com/640/480/business"
}

func (f CustomFaker) RandomCatsImage() string {
	return "http://placeimg.com/640/480/cats"
}

func (f CustomFaker) RandomCityImage() string {
	return "http://placeimg.com/640/480/city"
}

func (f CustomFaker) RandomFoodImage() string {
	return "http://placeimg.com/640/480/food"
}

func (f CustomFaker) RandomNightlifeImage() string {
	return "http://placeimg.com/640/480/nightlife"
}

func (f CustomFaker) RandomFashionImage() string {
	return "http://placeimg.com/640/480/fashion"
}

func (f CustomFaker) RandomPeopleImage() string {
	return "http://placeimg.com/640/480/people"
}

func (f CustomFaker) RandomNatureImage() string {
	return "http://placeimg.com/640/480/nature"
}

func (f CustomFaker) RandomSportsImage() string {
	return "http://placeimg.com/640/480/sports"
}

func (f CustomFaker) RandomTransportImage() string {
	return "http://placeimg.com/640/480/transport"
}

func (f CustomFaker) RandomCountryCode() string {
	return CountryCodes[f.Generator.Intn(len(CountryCodes))]
}

func (f CustomFaker) RandomPhoneNumber() string {
	return strconv.Itoa(f.Generator.Intn(999-100)+100) +
		"-" + strconv.Itoa(f.Generator.Intn(999-100)+100) +
		"-" + strconv.Itoa(f.Generator.Intn(9999-1000)+1000)
}

func (f CustomFaker) RandomPhoneNumberExt() string {
	return strconv.Itoa(f.Generator.Intn(99-10)+10) + "-" + f.RandomPhoneNumber()
}

func (f CustomFaker) RandomJobArea() string {
	return JobAreas[f.Generator.Intn(len(JobAreas))]
}

func (f CustomFaker) RandomJobDescriptor() string {
	return JobDescriptors[f.Generator.Intn(len(JobDescriptors))]
}

func (f CustomFaker) RandomJobType() string {
	return JobTypes[f.Generator.Intn(len(JobTypes))]
}

func (f CustomFaker) RandomJobTitle() string {
	return f.RandomJobDescriptor() + " " + f.RandomJobArea() + " " + f.RandomJobType()
}

func (f CustomFaker) RandomSemver() string {
	return strconv.Itoa(f.Generator.Intn(9)) +
		"." + strconv.Itoa(f.Generator.Intn(9)) +
		"." + strconv.Itoa(f.Generator.Intn(9))
}

func (f CustomFaker) RandomProtocol() string {
	return Protocols[f.Generator.Intn(len(Protocols))]
}

func (f CustomFaker) RandomAbbreviation() string {
	return Abbreviations[f.Generator.Intn(len(Abbreviations))]
}

func (f CustomFaker) RandomAlphanumeric() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, 1)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (f CustomFaker) Ipv6() string {
	ips := []string{}
	ipv6Alphabet := []string{
		"a", "b", "c", "d", "e", "f", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}

	for j := 0; j < 8; j++ {
		block := ""
		for w := 0; w < 4; w++ {
			block = block + ipv6Alphabet[rand.Intn(len(ipv6Alphabet))]
		}

		ips = append(ips, block)
	}

	return strings.Join(ips, ":")
}

func (f CustomFaker) RandomDigitNotNull() int {
	return f.Generator.Int()%8 + 1
}

// RandomDigit returns a fake random digit for Faker
func (f CustomFaker) RandomDigit() int {
	return f.Generator.Int() % 10
}

func (f CustomFaker) RandomFloat() float64 {
	return f.Generator.Float64()
}

func (f CustomFaker) RandomString() string {
	return f.RandomStringWithLength(10)
}

func (f CustomFaker) RandomStringWithLength(l int) string {
	r := []string{}
	for i := 0; i < l; i++ {
		r = append(r, f.RandomLetter())
	}
	return strings.Join(r, "")
}

func (f CustomFaker) RandomLetter() string {
	return fmt.Sprintf("%c", f.IntBetween(97, 122))
}

func (f CustomFaker) IntBetween(min, max int) int {
	diff := max - min

	if diff < 0 {
		diff = 0
	}

	if diff == 0 {
		return min
	}

	if diff == maxInt {
		return f.Generator.Intn(diff)
	}

	return f.Generator.Intn(diff+1) + min
}

// New returns a new instance of Faker instance with a random seed
func NewCustomFaker() (f CustomFaker) {
	seed := rand.NewSource(time.Now().Unix())
	f = NewWithSeed(seed)
	return
}

// NewWithSeed returns a new instance of Faker instance with a given seed
func NewWithSeed(src rand.Source) (f CustomFaker) {
	generator := rand.New(src)
	f = CustomFaker{Generator: generator}
	return
}
