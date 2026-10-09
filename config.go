package main

import "time"

// Tunables live here. DOM selectors are in selectors.go.

const (
	threadsURL   = "https://www.threads.com"
	searchURLFmt = threadsURL + "/search?q=%s&serp_type=default"

	profileDir = "threads_profile"
	outputDir  = "output"
	leadsFile  = "output/leads.json"
	rawFile    = "output/raw_posts.json"

	scrollsPerQuery = 4
	resultsTimeout  = 15 * time.Second
	loginTimeout    = 10 * time.Minute
)

var (
	scrollPauseMin = 1500 * time.Millisecond
	scrollPauseMax = 3000 * time.Millisecond
	queryPauseMin  = 3 * time.Second
	queryPauseMax  = 6 * time.Second
)

var searchQueriesRU = []string{
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

var searchQueriesEN = []string{
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

var searchQueries = append(append([]string{}, searchQueriesRU...), searchQueriesEN...)

var techKeywords = []string{
	"devops", "sre", "sysadmin", "системный администратор", "сисадмин",
	"developer", "разработчик", "разработчика", "программист", "программиста", "programmer",
	"engineer", "инженер",
	"backend", "бэкенд", "frontend", "фронтенд", "fullstack", "full-stack", "full stack",
	"python", "golang", "java", "node.js", "nodejs", "react", "vue", "django", "fastapi",
	"kubernetes", "k8s", "docker", "aws", "azure", "gcp", "terraform", "ansible",
	"ci/cd", "linux", "postgres", "postgresql",
}

var hiringKeywords = []string{
	// ru
	"ищу", "ищем", "нужен", "нужна", "нужны", "требуется", "требуются", "вакансия",
	"заказ", "проект", "подработка", "исполнитель", "на проект", "ищется",
	// en
	"hiring", "looking for", "need", "needed", "wanted", "contract", "freelance",
	"vacancy", "job opening", "project", "contractor", "seeking", "we are looking",
}

// Posts by people looking for a job themselves, not offering one.
var negativePatterns = []string{
	"ищу работу", "ищу удаленную работу", "ищу удалённую работу", "в поиске работы",
	"open to work", "opentowork", "looking for a job", "looking for job",
	"looking for work", "looking for new opportunities", "looking for opportunities",
	"my resume", "моё резюме", "мое резюме", "рассмотрю предложения",
	"available for hire", "available for work", "hire me",
}

// Order matters: first match wins.
var categoryRules = []struct {
	name  string
	words []string
}{
	{"freelance", []string{"freelance", "фриланс", "подработка", "разово", "разовая", "part-time", "парт-тайм"}},
	{"contract", []string{"contract", "контракт", "contractor", "b2b"}},
	{"job", []string{"вакансия", "vacancy", "job opening", "full-time", "фулл-тайм", "в штат", "hiring", "нанимаем"}},
	{"project", []string{"проект", "project", "заказ", "задача", "task", "mvp"}},
}
