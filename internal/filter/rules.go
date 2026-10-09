package filter

// Rule lists. A post is relevant when it mentions at least one tech term AND one hiring-intent term,
// and none of the negative patterns (people looking for a job themselves).

var TechKeywords = []string{
	"devops", "sre", "sysadmin", "системный администратор", "сисадмин",
	"developer", "разработчик", "разработчика", "программист", "программиста", "programmer",
	"engineer", "инженер",
	"backend", "бэкенд", "frontend", "фронтенд", "fullstack", "full-stack", "full stack",
	"python", "golang", "java", "node.js", "nodejs", "react", "vue", "django", "fastapi",
	"kubernetes", "k8s", "docker", "aws", "azure", "gcp", "terraform", "ansible",
	"ci/cd", "linux", "postgres", "postgresql",
}

var HiringKeywords = []string{
	// ru
	"ищу", "ищем", "нужен", "нужна", "нужны", "требуется", "требуются", "вакансия",
	"заказ", "проект", "подработка", "исполнитель", "на проект", "ищется",
	// en
	"hiring", "looking for", "need", "needed", "wanted", "contract", "freelance",
	"vacancy", "job opening", "project", "contractor", "seeking", "we are looking",
}

var NegativePatterns = []string{
	"ищу работу", "ищу удаленную работу", "ищу удалённую работу", "в поиске работы",
	"open to work", "opentowork", "looking for a job", "looking for job",
	"looking for work", "looking for new opportunities", "looking for opportunities",
	"my resume", "моё резюме", "мое резюме", "рассмотрю предложения",
	"available for hire", "available for work", "hire me",
}

// CategoryRules are checked in order; first match wins.
var CategoryRules = []struct {
	Name  string
	Words []string
}{
	{"freelance", []string{"freelance", "фриланс", "подработка", "разово", "разовая", "part-time", "парт-тайм"}},
	{"contract", []string{"contract", "контракт", "contractor", "b2b"}},
	{"job", []string{"вакансия", "vacancy", "job opening", "full-time", "фулл-тайм", "в штат", "hiring", "нанимаем"}},
	{"project", []string{"проект", "project", "заказ", "задача", "task", "mvp"}},
}
