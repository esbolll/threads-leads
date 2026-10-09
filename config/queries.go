package config

// Search queries. Deliberately phrased as hiring intent + role; bare "devops" / "developer" is too noisy.

var SearchQueriesRU = []string{
	"ищу devops", "нужен devops", "нужен devops инженер", "ищем devops",
	"нужен системный администратор", "нужен разработчик", "ищу разработчика",
	"ищем разработчика", "нужен backend разработчик", "нужен frontend разработчик",
	"нужен python разработчик", "нужен golang разработчик", "нужен программист",
	"ищу программиста", "нужен специалист kubernetes", "нужен специалист docker",
	"нужен специалист aws", "нужен специалист azure", "заказ разработка",
	"нужен разработчик на проект", "ищу разработчика на проект",
	"freelance разработчик", "подработка программист", "вакансия devops",
	"удаленная работа devops", "удаленная работа разработчик",
}

var SearchQueriesEN = []string{
	"hiring devops", "looking for devops", "need devops", "need a devops engineer",
	"looking for a devops engineer", "devops contract", "freelance devops",
	"devops freelance", "devops needed", "hiring developer", "looking for developer",
	"need developer", "looking for backend developer", "looking for frontend developer",
	"looking for python developer", "looking for golang developer",
	"looking for software engineer", "developer needed", "engineer needed",
	"looking for kubernetes engineer", "kubernetes contract", "aws engineer needed",
	"azure engineer needed", "hiring remote developer", "remote devops job",
	"freelance developer", "software project developer needed",
}

func SearchQueries() []string {
	out := make([]string, 0, len(SearchQueriesRU)+len(SearchQueriesEN))
	out = append(out, SearchQueriesRU...)
	return append(out, SearchQueriesEN...)
}
