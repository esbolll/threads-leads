package config

// Search queries, grouped by priority: development orders first, then DevOps, then turnkey sites/bots/apps.
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

var QueriesDevEN = []string{
	"hiring developer", "looking for developer", "need developer", "developer needed",
	"looking for a developer", "need a developer", "looking for software engineer", "engineer needed",
	"looking for backend developer", "looking for frontend developer", "looking for fullstack developer",
	"looking for python developer", "looking for golang developer", "looking for php developer",
	"freelance developer", "contract developer", "hiring remote developer",
	"software project developer needed", "need a programmer",
}

var QueriesDevOpsRU = []string{
	"ищу devops", "нужен devops", "нужен devops инженер", "ищем devops",
	"нужен системный администратор", "ищу сисадмина",
	"нужен специалист kubernetes", "нужен специалист docker",
	"нужен специалист aws", "нужен специалист azure",
	"вакансия devops", "удаленная работа devops", "настроить сервер",
}

var QueriesDevOpsEN = []string{
	"hiring devops", "looking for devops", "need devops", "need a devops engineer",
	"looking for a devops engineer", "devops contract", "freelance devops", "devops needed",
	"looking for kubernetes engineer", "kubernetes contract",
	"aws engineer needed", "azure engineer needed", "remote devops job", "sysadmin needed",
}

var QueriesWebRU = []string{
	"сайт под ключ", "нужен сайт", "сделать сайт", "разработка сайта", "заказать сайт",
	"нужен лендинг", "сделать лендинг", "нужен интернет магазин",
	"нужен wordpress разработчик", "ищу wordpress разработчика",
	"нужен телеграм бот", "сделать телеграм бота", "разработка бота",
	"нужно мобильное приложение", "разработка приложения", "сделать приложение",
	"нужен верстальщик", "нужна интеграция", "нужна автоматизация",
}

var QueriesWebEN = []string{
	"need a website", "website developer needed", "build me a website", "looking for web developer",
	"landing page developer", "wordpress developer needed", "looking for wordpress developer",
	"need a telegram bot", "bot developer needed", "need an app developed",
	"mobile app developer needed", "looking for app developer", "need automation developer",
}

func SearchQueries() []string {
	groups := [][]string{
		QueriesDevRU, QueriesDevEN,
		QueriesDevOpsRU, QueriesDevOpsEN,
		QueriesWebRU, QueriesWebEN,
	}
	var out []string
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}
