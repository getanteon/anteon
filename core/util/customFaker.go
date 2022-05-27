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
		"Paanga", "Turkish Lira", "Trinidad and Tobago Dollar", "New Taiwan Dollar", "Tanzanian Shilling", "Hryvnia",
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

	CompanySuffixes = []string{"AG", "GmbH", "und Söhne", "und Partner", "& Co.", "Gruppe", "LLC", "Inc."}

	BusinessVerbs = []string{
		"implement", "utilize", "integrate", "streamline", "optimize", "evolve", "transform", "embrace", "enable",
		"orchestrate", "leverage", "reinvent", "aggregate", "architect", "enhance", "incentivize", "morph", "empower",
		"envisioneer", "monetize", "harness", "facilitate", "seize", "disintermediate", "synergize", "strategize", "deploy",
		"brand", "grow", "target", "syndicate", "synthesize", "deliver", "mesh", "incubate", "engage", "maximize",
		"benchmark", "expedite", "reintermediate", "whiteboard", "visualize", "repurpose", "innovate", "scale", "unleash",
		"drive", "extend", "engineer", "revolutionize", "generate", "exploit", "transition", "e-enable", "iterate",
		"cultivate", "matrix", "productize", "redefine", "recontextualize",
	}

	BusinessAdjectives = []string{
		"clicks-and-mortar", "value-added", "vertical", "proactive", "robust", "revolutionary", "scalable", "leading-edge",
		"innovative", "intuitive", "strategic", "e-business", "mission-critical", "sticky", "one-to-one", "24/7",
		"end-to-end", "global", "B2B", "B2C", "granular", "frictionless", "virtual", "viral", "dynamic", "24/365",
		"best-of-breed", "killer", "magnetic", "bleeding-edge", "web-enabled", "interactive", "dot-com", "sexy", "back-end",
		"real-time", "efficient", "front-end", "distributed", "seamless", "extensible", "turn-key", "world-class",
		"open-source", "cross-platform", "cross-media", "synergistic", "bricks-and-clicks", "out-of-the-box", "enterprise",
		"integrated", "impactful", "wireless", "transparent", "next-generation", "cutting-edge", "user-centric", "visionary",
		"customized", "ubiquitous", "plug-and-play", "collaborative", "compelling", "holistic", "rich",
	}

	BusinessNouns = []string{
		"synergies", "web-readiness", "paradigms", "markets", "partnerships", "infrastructures", "platforms", "initiatives",
		"channels", "eyeballs", "communities", "ROI", "solutions", "e-tailers", "e-services", "action-items", "portals",
		"niches", "technologies", "content", "vortals", "supply-chains", "convergence", "relationships", "architectures",
		"interfaces", "e-markets", "e-commerce", "systems", "bandwidth", "infomediaries", "models", "mindshare",
		"deliverables", "users", "schemas", "networks", "applications", "metrics", "e-business", "functionalities",
		"experiences", "web services", "methodologies", "blockchains",
	}

	CompanyAdjectives = []string{
		"Adaptive", "Advanced", "Ameliorated", "Assimilated", "Automated", "Balanced", "Business-focused", "Centralized",
		"Cloned", "Compatible", "Configurable", "Cross-group", "Cross-platform", "Customer-focused", "Customizable",
		"Decentralized", "De-engineered", "Devolved", "Digitized", "Distributed", "Diverse", "Down-sized", "Enhanced",
		"Enterprise-wide", "Ergonomic", "Exclusive", "Expanded", "Extended", "Face to face", "Focused", "Front-line",
		"Fully-configurable", "Function-based", "Fundamental", "Future-proofed", "Grass-roots", "Horizontal", "Implemented",
		"Innovative", "Integrated", "Intuitive", "Inverse", "Managed", "Mandatory", "Monitored", "Multi-channeled",
		"Multi-lateral", "Multi-layered", "Multi-tiered", "Networked", "Object-based", "Open-architected", "Open-source",
		"Operative", "Optimized", "Optional", "Organic", "Organized", "Persevering", "Persistent", "Phased", "Polarized",
		"Pre-emptive", "Proactive", "Profit-focused", "Profound", "Programmable", "Progressive", "Public-key",
		"Quality-focused", "Reactive", "Realigned", "Re-contextualized", "Re-engineered", "Reduced", "Reverse-engineered",
		"Right-sized", "Robust", "Seamless", "Secured", "Self-enabling", "Sharable", "Stand-alone", "Streamlined",
		"Switchable", "Synchronized", "Synergistic", "Synergized", "Team-oriented", "Total", "Triple-buffered", "Universal",
		"Up-sized", "Upgradable", "User-centric", "User-friendly", "Versatile", "Virtual", "Visionary", "Vision-oriented",
	}

	CompanyDescriptors = []string{
		"24 hour", "24/7", "3rd generation", "4th generation", "5th generation", "6th generation", "actuating", "analyzing",
		"asymmetric", "asynchronous", "attitude-oriented", "background", "bandwidth-monitored", "bi-directional",
		"bifurcated", "bottom-line", "clear-thinking", "client-driven", "client-server", "coherent", "cohesive", "composite",
		"context-sensitive", "contextually-based", "content-based", "dedicated", "demand-driven", "didactic", "directional",
		"discrete", "disintermediate", "dynamic", "eco-centric", "empowering", "encompassing", "even-keeled", "executive",
		"explicit", "exuding", "fault-tolerant", "foreground", "fresh-thinking", "full-range", "global", "grid-enabled",
		"heuristic", "high-level", "holistic", "homogeneous", "human-resource", "hybrid", "impactful", "incremental",
		"intangible", "interactive", "intermediate", "leading edge", "local", "logistical", "maximized", "methodical",
		"mission-critical", "mobile", "modular", "motivating", "multimedia", "multi-state", "multi-tasking", "national",
		"needs-based", "neutral", "next generation", "non-volatile", "object-oriented", "optimal", "optimizing", "radical",
		"real-time", "reciprocal", "regional", "responsive", "scalable", "secondary", "solution-oriented", "stable", "static",
		"systematic", "systemic", "system-worthy", "tangible", "tertiary", "transitional", "uniform", "upward-trending",
		"user-facing", "value-added", "web-enabled", "well-modulated", "zero administration", "zero defect", "zero tolerance",
	}

	CompanyNouns = []string{
		"ability", "access", "adapter", "algorithm", "alliance", "analyzer", "application", "approach", "architecture",
		"archive", "artificial intelligence", "array", "attitude", "benchmark", "budgetary management", "capability",
		"capacity", "challenge", "circuit", "collaboration", "complexity", "concept", "conglomeration", "contingency", "core",
		"customer loyalty", "database", "data-warehouse", "definition", "emulation", "encoding", "encryption", "extranet",
		"firmware", "flexibility", "focus group", "forecast", "frame", "framework", "function", "functionalities",
		"Graphic Interface", "groupware", "Graphical User Interface", "hardware", "help-desk", "hierarchy", "hub",
		"implementation", "info-mediaries", "infrastructure", "initiative", "installation", "instruction set", "interface",
		"internet solution", "intranet", "knowledge user", "knowledge base", "local area network", "leverage", "matrices",
		"matrix", "methodology", "middleware", "migration", "model", "moderator", "monitoring", "moratorium", "neural-net",
		"open architecture", "open system", "orchestration", "paradigm", "parallelism", "policy", "portal",
		"pricing structure", "process improvement", "product", "productivity", "project", "projection", "protocol",
		"secured line", "service-desk", "software", "solution", "standardization", "strategy", "structure", "success",
		"superstructure", "support", "synergy", "system engine", "task-force", "throughput", "time-frame", "toolset",
		"utilization", "website", "workforce",
	}

	DatabaseColumns = []string{
		"id", "title", "name", "email", "phone", "token", "group", "category", "password", "comment", "avatar", "status",
		"createdAt", "updatedAt",
	}

	DatabaseTypes = []string{
		"int", "varchar", "text", "date", "datetime", "tinyint", "time", "timestamp", "smallint", "mediumint", "bigint",
		"decimal", "float", "double", "real", "bit", "boolean", "serial", "blob", "binary", "enum", "set", "geometry",
		"point",
	}

	DatabaseCollations = []string{
		"utf8_unicode_ci", "utf8_general_ci", "utf8_bin", "ascii_bin", "ascii_general_ci", "cp1250_bin", "cp1250_general_ci",
	}

	DatabaseEngines = []string{"InnoDB", "MyISAM", "MEMORY", "CSV", "BLACKHOLE", "ARCHIVE"}

	WeekDays = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	Months = []string{
		"January", "February", "March", "April", "May", "June", "July", "August", "September",
		"October", "November", "December",
	}

	DomainSuffixes = []string{
		"ac.uk", "biz", "co", "co.uk", "com", "cymru", "gov.uk", "info", "london", "ltd.uk", "me.uk", "name", "nhs.uk",
		"org.uk", "plc.uk", "sch.uk", "scot", "uk", "wales", "com.tr",
	}

	FirstNames = []string{
		"William", "Jack", "Oliver", "Joshua", "Thomas", "Lachlan", "Cooper", "Noah", "Ethan", "Lucas", "James", "Samuel",
		"Jacob", "Liam", "Alexander", "Benjamin", "Max", "Isaac", "Daniel", "Riley", "Ryan", "Charlie", "Tyler", "Jake",
		"Matthew", "Xavier", "Harry", "Jayden", "Nicholas", "Harrison", "Levi", "Luke", "Adam", "Henry", "Aiden", "Dylan",
		"Oscar", "Michael", "Jackson", "Logan", "Joseph", "Blake", "Nathan", "Connor", "Elijah", "Nate", "Archie", "Bailey",
		"Marcus", "Cameron", "Jordan", "Zachary", "Caleb", "Hunter", "Ashton", "Toby", "Aidan", "Hayden", "Mason", "Hamish",
		"Edward", "Angus", "Eli", "Sebastian", "Christian", "Patrick", "Andrew", "Anthony", "Luca", "Kai", "Beau", "Alex",
		"George", "Callum", "Finn", "Zac", "Mitchell", "Jett", "Jesse", "Gabriel", "Leo", "Declan", "Charles", "Jasper",
		"Jonathan", "Aaron", "Hugo", "David", "Christopher", "Chase", "Owen", "Justin", "Ali", "Darcy", "Lincoln", "Cody",
		"Phoenix", "Sam", "John", "Joel", "Isabella", "Ruby", "Chloe", "Olivia", "Charlotte", "Mia", "Lily", "Emily",
		"Ella", "Sienna", "Sophie", "Amelia", "Grace", "Ava", "Zoe", "Emma", "Sophia", "Matilda", "Hannah", "Jessica",
		"Lucy", "Georgia", "Sarah", "Abigail", "Zara", "Eva", "Scarlett", "Jasmine", "Chelsea", "Lilly", "Ivy", "Isla",
		"Evie", "Isabelle", "Maddison", "Layla", "Summer", "Annabelle", "Alexis", "Elizabeth", "Bella", "Holly", "Lara",
		"Madison", "Alyssa", "Maya", "Tahlia", "Claire", "Hayley", "Imogen", "Jade", "Ellie", "Sofia", "Addison", "Kiara",
		"Molly", "Phoebe", "Alice", "Savannah", "Gabriella", "Kayla", "Mikayla", "Abbey", "Eliza", "Willow", "Alexandra",
		"Poppy", "Samantha", "Stella", "Amy", "Amelie", "Anna", "Piper", "Gemma", "Isabel", "Victoria", "Stephanie",
		"Caitlin", "Heidi", "Paige", "Rose", "Amber", "Audrey", "Claudia", "Taylor", "Madeline", "Angelina", "Natalie",
		"Charli", "Lauren", "Ashley", "Violet", "Mackenzie", "Abby", "Skye", "Lillian", "Alana", "Lola", "Leah", "Eve",
	}

	LastNames = []string{
		"Smith", "Jones", "Williams", "Brown", "Wilson", "Taylor", "Johnson", "White", "Martin", "Anderson", "Thompson",
		"Nguyen", "Thomas", "Walker", "Harris", "Lee", "Ryan", "Robinson", "Kelly", "King", "Davis", "Wright", "Evans",
		"Roberts", "Green", "Hall", "Wood", "Jackson", "Clarke", "Patel", "Khan", "Lewis", "James", "Phillips", "Mason",
		"Mitchell", "Rose", "Davies", "Rodriguez", "Cox", "Alexander", "Garden", "Campbell", "Johnston", "Moore", "Smyth",
		"Oneill", "Doherty", "Stewart", "Quinn", "Murphy", "Graham", "Mclaughlin", "Hamilton", "Murray", "Hughes",
		"Robertson", "Thomson", "Scott", "Macdonald", "Reid", "Clark", "Ross", "Young", "Watson", "Paterson", "Morrison",
		"Morgan", "Griffiths", "Edwards", "Rees", "Jenkins", "Owen", "Price", "Moss", "Richards", "Abbott", "Adams",
		"Armstrong", "Bahringer", "Bailey", "Barrows", "Bartell", "Bartoletti", "Barton", "Bauch", "Baumbach", "Bayer",
		"Beahan", "Beatty", "Becker", "Beier", "Berge", "Bergstrom", "Bode", "Bogan", "Borer", "Bosco", "Botsford", "Boyer",
		"Boyle", "Braun", "Bruen", "Carroll", "Carter", "Cartwright", "Casper", "Cassin", "Champlin", "Christiansen",
		"Cole", "Collier", "Collins", "Connelly", "Conroy", "Corkery", "Cormier", "Corwin", "Cronin", "Crooks", "Cruickshank",
		"Cummings", "Damore", "Daniel", "Dare", "Daugherty", "Dickens", "Dickinson", "Dietrich", "Donnelly", "Dooley",
		"Douglas", "Doyle", "Durgan", "Ebert", "Emard", "Emmerich", "Erdman", "Ernser", "Fadel", "Fahey", "Farrell", "Fay",
		"Feeney", "Feil", "Ferry", "Fisher", "Flatley", "Gibson", "Gleason", "Glover", "Goldner", "Goodwin", "Grady", "Grant",
		"Greenfelder", "Greenholt", "Grimes", "Gutmann", "Hackett", "Hahn", "Haley", "Hammes", "Hand", "Hane", "Hansen",
		"Harber", "Hartmann", "Harvey", "Hayes", "Heaney", "Heathcote", "Heller", "Hermann", "Hermiston", "Hessel",
		"Hettinger", "Hickle", "Hill", "Hills", "Hoppe", "Howe", "Howell", "Hudson", "Huel", "Hyatt", "Jacobi", "Jacobs",
		"Jacobson", "Jerde", "Johns", "Keeling", "Kemmer", "Kessler", "Kiehn", "Kirlin", "Klein", "Koch", "Koelpin",
		"Kohler", "Koss", "Kovacek", "Kreiger", "Kris", "Kuhlman", "Kuhn", "Kulas", "Kunde", "Kutch", "Lakin", "Lang",
		"Langworth", "Larkin", "Larson", "Leannon", "Leffler", "Little", "Lockman", "Lowe", "Lynch", "Mann", "Marks",
		"Marvin", "Mayer", "Mccullough", "Mcdermott", "Mckenzie", "Miller", "Mills", "Monahan", "Morissette", "Mueller",
		"Muller", "Nader", "Nicolas", "Nolan", "Oconnell", "Oconner", "Ohara", "Okeefe", "Olson", "Oreilly", "Parisian",
		"Parker", "Quigley", "Reilly", "Reynolds", "Rice", "Ritchie", "Rohan", "Rolfson", "Rowe", "Russel", "Rutherford",
		"Sanford", "Sauer", "Schmidt", "Schmitt", "Schneider", "Schroeder", "Schultz", "Shields", "Smitham", "Spencer",
		"Stanton", "Stark", "Stokes", "Swift", "Tillman", "Towne", "Tremblay", "Tromp", "Turcotte", "Turner", "Walsh",
		"Walter", "Ward", "Waters", "Weber", "Welch", "West", "Wilderman", "Wilkinson", "Williamson", "Windler", "Wolf",
	}

	Emails = []string{"gmail.com", "yahoo.com", "hotmail.com", "protonmail.com", "outlook.com", "hey.com"}

	ExampleEmails = []string{"example.org", "example.com", "example.net"}

	DirectoryPaths = []string{
		"/Applications", "/bin", "/boot", "/boot/defaults", "/dev", "/etc", "/etc/defaults", "/etc/mail", "/etc/namedb",
		"/etc/periodic", "/etc/ppp", "/home", "/home/user", "/home/user/dir", "/lib", "/Library", "/lost+found", "/media",
		"/mnt", "/net", "/Network", "/opt", "/opt/bin", "/opt/include", "/opt/lib", "/opt/sbin", "/opt/share", "/private",
		"/private/tmp", "/private/var", "/proc", "/rescue", "/root", "/sbin", "/selinux", "/srv", "/sys", "/System", "/tmp",
		"/Users", "/usr", "/usr/X11R6", "/usr/bin", "/usr/include", "/usr/lib", "/usr/libdata", "/usr/libexec",
		"/usr/local/src", "/usr/obj", "/usr/ports", "/usr/sbin", "/usr/share", "/usr/src", "/var", "/var/log", "/var/mail",
		"/var/spool", "/var/tmp", "/var/yp", "/usr/local/bin",
	}

	FileExtensions = []string{
		"png", "gif", "doc", "docx", "pdf", "xls", "ppt", "jpg", "mp4", "mov", "wav", "html", "json", "xml", "tif", "tsv",
		"csv", "gdoc", "lzh", "wsdl", "html", "war", "book", "fsc",
	}

	CommonFileExtensions = []string{
		"pdf", "mpeg", "wav", "png", "jpeg", "gif", "mp4", "html", "m3a",
	}

	FileTypes = []string{"video", "audio", "image", "text", "application", "multipart", "model"}

	CommonFileTypes = []string{"video", "audio", "image", "text", "application"}

	CommonMimeTypes = []string{
		"application/pdf", "audio/mpeg", "audio/wav", "image/png", "image/jpeg", "image/gif", "video/mp4", "video/mpeg",
		"text/html",
	}

	Products = []string{
		"Chair", "Car", "Computer", "Keyboard", "Mouse", "Bike", "Ball", "Gloves", "Pants", "Shirt", "Table", "Shoes", "Hat",
		"Towels", "Soap", "Tuna", "Chicken", "Fish", "Cheese", "Bacon", "Pizza", "Salad", "Sausages", "Chips",
	}

	ProductMaterials = []string{
		"Steel", "Bronze", "Wooden", "Concrete", "Plastic", "Cotton", "Granite", "Rubber", "Metal", "Soft", "Fresh", "Frozen",
	}

	ProductAdjectives = []string{
		"Small", "Ergonomic", "Electronic", "Rustic", "Intelligent", "Gorgeous", "Incredible", "Elegant", "Fantastic",
		"Practical", "Modern", "Recycled", "Sleek", "Bespoke", "Awesome", "Generic", "Handcrafted", "Handmade", "Oriental",
		"Licensed", "Luxurious", "Refined", "Unbranded", "Tasty",
	}

	StoreDepartments = []string{
		"Books", "Movies", "Music", "Games", "Electronics", "Computers", "Home", "Garden", "Tools", "Grocery", "Health",
		"Beauty", "Toys", "Kids", "Baby", "Clothing", "Shoes", "Jewelery", "Sports", "Outdoors", "Automotive", "Industrial",
	}

	Nouns = []string{
		"driver", "protocol", "bandwidth", "panel", "microchip", "program", "port", "card", "array", "interface", "system",
		"sensor", "firewall", "hard drive", "pixel", "alarm", "feed", "monitor", "application", "transmitter", "bus",
		"circuit", "capacitor", "matrix",
	}

	Verbs = []string{
		"back up", "bypass", "hack", "override", "compress", "copy", "navigate", "index", "connect", "generate", "quantify",
		"calculate", "synthesize", "input", "transmit", "program", "reboot", "parse",
	}

	IngVerbs = []string{
		"backing up", "bypassing", "hacking", "overriding", "compressing", "copying", "navigating", "indexing", "connecting",
		"generating", "quantifying", "calculating", "synthesizing", "transmitting", "programming", "parsing",
	}

	Adjectives = []string{
		"auxiliary", "primary", "back-end", "digital", "open-source", "virtual", "cross-platform", "redundant", "online",
		"haptic", "multi-byte", "bluetooth", "wireless", "1080p", "neural", "optical", "solid state", "mobile",
	}
)

type CustomFaker struct {
	Generator *rand.Rand
}

func (f CustomFaker) RandomNoun() string {
	return Nouns[f.Generator.Intn(len(Nouns))]
}

func (f CustomFaker) RandomVerb() string {
	return Verbs[f.Generator.Intn(len(Verbs))]
}

func (f CustomFaker) RandomIngVerb() string {
	return IngVerbs[f.Generator.Intn(len(IngVerbs))]
}

func (f CustomFaker) RandomAdjective() string {
	return Adjectives[f.Generator.Intn(len(Adjectives))]
}

func (f CustomFaker) RandomWord() string {
	words := append(append(append(Adjectives, IngVerbs...), Verbs...), Nouns...)
	return words[f.Generator.Intn(len(words))]
}

func (f CustomFaker) RandomDepartment() string {
	return StoreDepartments[f.Generator.Intn(len(StoreDepartments))]
}

func (f CustomFaker) RandomProductName() string {
	return f.RandomProductAdjective() + " " + f.RandomProductMaterial() + " " + f.RandomProduct()
}

func (f CustomFaker) RandomProductMaterial() string {
	return ProductMaterials[f.Generator.Intn(len(ProductMaterials))]
}

func (f CustomFaker) RandomProductAdjective() string {
	return ProductAdjectives[f.Generator.Intn(len(ProductAdjectives))]
}

func (f CustomFaker) RandomProduct() string {
	return Products[f.Generator.Intn(len(Products))]
}

func (f CustomFaker) RandomPrice() string {
	return strconv.Itoa(f.Generator.Intn(1000)) + "." + strconv.Itoa(f.Generator.Intn(99-10)+10)
}

func (f CustomFaker) RandomFilePath() string {
	return f.RandomDirectoryPath() + "/" + f.RandomDomainWord() + "." + f.RandomFileExtension()
}

func (f CustomFaker) RandomMimeTypes() string {
	return CommonMimeTypes[f.Generator.Intn(len(CommonMimeTypes))]
}

func (f CustomFaker) RandomDirectoryPath() string {
	return DirectoryPaths[f.Generator.Intn(len(DirectoryPaths))]
}

func (f CustomFaker) RandomCommonFileExtension() string {
	return CommonFileExtensions[f.Generator.Intn(len(CommonFileExtensions))]
}

func (f CustomFaker) RandomCommonFileType() string {
	return CommonFileTypes[f.Generator.Intn(len(CommonFileTypes))]
}

func (f CustomFaker) RandomCommonFileName() string {
	return f.RandomDomainWord() + "." + f.RandomFileExtension()
}

func (f CustomFaker) RandomFileType() string {
	return FileTypes[f.Generator.Intn(len(FileTypes))]
}

func (f CustomFaker) RandomFileExtension() string {
	return FileExtensions[f.Generator.Intn(len(FileExtensions))]
}

func (f CustomFaker) RandomFileName() string {
	return strings.ToLower(FirstNames[f.Generator.Intn(len(FirstNames))]+"_"+
		LastNames[f.Generator.Intn(len(LastNames))]) + "." + f.RandomFileExtension()
}

func (f CustomFaker) RandomUrl() string {
	return f.RandomProtocol() + "://" + strings.ToLower(FirstNames[f.Generator.Intn(len(FirstNames))]+
		LastNames[f.Generator.Intn(len(LastNames))]) + "." + f.RandomDomainSuffix()
}

func (f CustomFaker) RandomUsername() string {
	return FirstNames[f.Generator.Intn(len(FirstNames))] + "." + LastNames[f.Generator.Intn(len(LastNames))]
}

func (f CustomFaker) RandomExampleEmail() string {
	return strings.ToLower(FirstNames[f.Generator.Intn(len(FirstNames))]+"."+
		LastNames[f.Generator.Intn(len(LastNames))]) + "@" + ExampleEmails[f.Generator.Intn(len(ExampleEmails))]
}

func (f CustomFaker) RandomEmail() string {
	return strings.ToLower(FirstNames[f.Generator.Intn(len(FirstNames))]+"."+
		LastNames[f.Generator.Intn(len(LastNames))]) + "@" + Emails[f.Generator.Intn(len(Emails))]
}

func (f CustomFaker) RandomDomainWord() string {
	return strings.ToLower(FirstNames[f.Generator.Intn(len(FirstNames))] + LastNames[f.Generator.Intn(len(LastNames))])
}

func (f CustomFaker) RandomDomainSuffix() string {
	return DomainSuffixes[f.Generator.Intn(len(DomainSuffixes))]
}

func (f CustomFaker) RandomDomainName() string {
	return strings.ToLower(FirstNames[f.Generator.Intn(len(FirstNames))]+LastNames[f.Generator.Intn(len(LastNames))]) +
		"." + f.RandomDomainSuffix()
}

func (f CustomFaker) RandomMonth() string {
	return Months[f.Generator.Intn(len(Months))]
}

func (f CustomFaker) RandomWeekday() string {
	return WeekDays[f.Generator.Intn(len(WeekDays))]
}

func (f CustomFaker) RandomDatabaseColumn() string {
	return DatabaseColumns[f.Generator.Intn(len(DatabaseColumns))]
}

func (f CustomFaker) RandomDatabaseType() string {
	return DatabaseTypes[f.Generator.Intn(len(DatabaseTypes))]
}

func (f CustomFaker) RandomDatabaseCollation() string {
	return DatabaseCollations[f.Generator.Intn(len(DatabaseCollations))]
}

func (f CustomFaker) RandomDatabaseEngine() string {
	return DatabaseEngines[f.Generator.Intn(len(DatabaseEngines))]
}

func (f CustomFaker) RandomCatchPhrases() string {
	return f.RandomCatchPhraseAdjectives() + " " + f.RandomCatchPhraseDescriptors() + " " + f.RandomCatchPhraseNouns()
}

func (f CustomFaker) RandomCatchPhraseAdjectives() string {
	return CompanyAdjectives[f.Generator.Intn(len(CompanyAdjectives))]
}

func (f CustomFaker) RandomCatchPhraseDescriptors() string {
	return CompanyDescriptors[f.Generator.Intn(len(CompanyDescriptors))]
}

func (f CustomFaker) RandomCatchPhraseNouns() string {
	return CompanyNouns[f.Generator.Intn(len(CompanyNouns))]
}

func (f CustomFaker) RandomBsNouns() string {
	return BusinessNouns[f.Generator.Intn(len(BusinessNouns))]
}

func (f CustomFaker) RandomBsBuzzVerbs() string {
	return BusinessVerbs[f.Generator.Intn(len(BusinessVerbs))]
}

func (f CustomFaker) RandomBsAdjective() string {
	return BusinessAdjectives[f.Generator.Intn(len(BusinessAdjectives))]
}

func (f CustomFaker) RandomBs() string {
	return f.RandomBsBuzzVerbs() + " " + f.RandomBsAdjective() + " " + f.RandomBsNouns()
}

func (f CustomFaker) RandomCompanySuffix() string {
	return CompanySuffixes[f.Generator.Intn(len(CompanySuffixes))]
}

func (f CustomFaker) RandomBitcoin() string {
	const letters = "123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

	b := make([]byte, f.Generator.Intn(35-26)+26)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (f CustomFaker) RandomCurrencySymbol() string {
	return CurrencySymbols[f.Generator.Intn(len(CurrencySymbols))]
}

func (f CustomFaker) RandomCurrencyCode() string {
	return CurrencyCodes[f.Generator.Intn(len(CurrencyCodes))]
}

func (f CustomFaker) RandomCurrencyName() string {
	return CurrencyNames[f.Generator.Intn(len(CurrencyNames))]
}

func (f CustomFaker) RandomTransactionType() string {
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
