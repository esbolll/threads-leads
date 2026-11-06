package config

// Search queries, grouped by priority: development orders first, then DevOps, then turnkey sites/bots/apps.
// Target market is Russian- and Kazakh-speaking; English queries are pinned to the region by a city or country,
// because the region filter drops English posts without a CIS marker anyway.
// Phrased as hiring intent + role; bare "developer" / "devops" is too noisy.

var QueriesDevRU = []string{
	"нужен разработчик", "ищу разработчика", "ищем разработчика",
	"нужен разработчик на проект", "ищу разработчика на проект",
	"нужен программист", "ищу программиста", "ищем программиста",
	"нужен backend разработчик", "нужен frontend разработчик", "нужен fullstack разработчик",
	"нужен python разработчик", "нужен golang разработчик", "нужен php разработчик",
	"заказ разработка", "заказать разработку", "разработка на заказ",
	"freelance разработчик", "подработка программист", "подработка разработчик",
	"удаленная работа разработчик", "вакансия разработчик",
}

var QueriesDevOpsRU = []string{
	"ищу devops", "нужен devops", "нужен devops инженер", "ищем devops",
	"нужен системный администратор", "ищу сисадмина",
	"нужен специалист kubernetes", "нужен специалист docker",
	"нужен специалист aws", "нужен специалист azure",
	"вакансия devops", "удаленная работа devops", "настроить сервер",
}

var QueriesWebRU = []string{
	"сайт под ключ", "нужен сайт", "сделать сайт", "разработка сайта", "заказать сайт",
	"нужен лендинг", "сделать лендинг", "нужен интернет магазин",
	"нужен wordpress разработчик", "ищу wordpress разработчика",
	"нужен телеграм бот", "сделать телеграм бота", "разработка бота",
	"нужно мобильное приложение", "разработка приложения", "сделать приложение",
	"нужен верстальщик", "нужна интеграция", "нужна автоматизация",
}

var QueriesKK = []string{
	"бағдарламашы керек", "әзірлеуші керек", "бағдарламашы іздеймін", "әзірлеуші іздейміз",
	"сайт керек", "сайт жасау керек", "сайт жасап беретін", "лендинг керек",
	"бот жасау керек", "телеграм бот керек", "қосымша жасау керек", "автоматизация керек",
}

// English, pinned to the region.
var QueriesENRegion = []string{
	"developer Kazakhstan", "developer Almaty", "developer Astana", "hiring developer Kazakhstan",
	"need developer Almaty", "devops Kazakhstan", "devops Almaty", "website Almaty", "need a website Kazakhstan",
	"developer Tashkent", "developer Uzbekistan", "developer Bishkek", "developer CIS", "hiring Almaty IT",
}

func SearchQueries() []string {
	groups := [][]string{
		QueriesDevRU, QueriesDevOpsRU, QueriesWebRU, QueriesKK, QueriesENRegion,
	}
	var out []string
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}
