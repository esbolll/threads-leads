package filter

// Rule lists. A post is a lead when a hiring word sits next to a tech word:
//   - role words (StrongTech: "разработчик", "devops") are enough on their own;
//   - thing words (WeakTech: "сайт", "бот", "app") are everywhere, so they also need a marker that
//     the author is looking for a person or has a budget (SeekMarkers).
// NegativePatterns reject job seekers, sellers and off-topic "engineer" posts.

// StrongTech names a specialist or a stack: someone asking for one of these is hiring.
var StrongTech = []string{
	"devops", "sre", "sysadmin", "системный администратор", "системного администратора", "сисадмин", "сисадмина",
	"developer", "developers", "разработчик", "разработчика", "разработчики", "разработчиков",
	"программист", "программиста", "программисты", "программистов", "programmer", "programmers",
	"engineer", "engineers", "инженер", "инженера",
	"backend", "бэкенд", "frontend", "фронтенд", "fullstack", "full-stack", "full stack",
	"верстальщик", "верстальщика", "вебмастер", "веб-разработчик", "веб разработчик",
	"python", "golang", "java", "node.js", "nodejs", "react", "vue", "django", "fastapi", "php", "laravel",
	"flutter", "kotlin", "swift",
	"kubernetes", "k8s", "docker", "aws", "azure", "gcp", "terraform", "ansible",
	"ci/cd", "linux", "postgres", "postgresql",
	// kazakh
	"бағдарламашы", "әзірлеуші",
}

// WeakTech names a deliverable. Common in everyday talk, so it needs a SeekMarker as well.
var WeakTech = []string{
	"сайт", "сайта", "сайты", "сайтты", "сайтын", "лендинг", "landing", "website", "web", "wordpress",
	"tilda", "тильда", "битрикс", "bitrix", "интернет-магазин", "интернет магазин", "верстка", "вёрстка",
	"бот", "бота", "bot", "telegram", "телеграм", "чат-бот", "чат бот",
	"приложение", "приложения", "app", "mobile", "ios", "android",
	"интеграция", "интеграцию", "автоматизация", "автоматизацию", "automation", "api", "crm",
	"парсер", "parser", "қосымша",
}

// TechKeywords is everything, used for display.
var TechKeywords = append(append([]string{}, StrongTech...), WeakTech...)

// HiringKeywords must appear close to a tech word.
var HiringKeywords = []string{
	// ru
	"ищу", "ищем", "нужен", "нужна", "нужны", "нужно", "требуется", "требуются", "вакансия",
	"подработка", "ищется", "в поиске", "разыскивается", "кто сделает", "кто может сделать",
	// kk
	"керек", "қажет", "іздеймін", "іздейміз", "іздеп",
	// en
	"hiring", "looking for", "need", "needed", "wanted", "seeking", "we are looking",
	"freelance", "contract", "vacancy", "job opening",
}

// SeekMarkers show that the author wants a person to do the work, or has money for it.
var SeekMarkers = []string{
	// ru
	"ищу", "ищем", "кто сделает", "кто может", "кто делает", "кто умеет", "кто-нибудь", "кто нибудь",
	"посоветуйте", "порекомендуйте", "подскажите специалиста", "специалист", "специалиста",
	"исполнитель", "исполнителя", "фрилансер", "фрилансера", "подрядчик", "подрядчика",
	"срочно", "бюджет", "оплата", "заплачу", "готов заплатить", "готова заплатить", "за деньги",
	"сколько стоит", "сколько будет стоить", "цена вопроса", "тз", "техзадание", "тех задание",
	"пишите в директ", "пишите в лс", "пишите в личку", "напишите в директ", "в личные сообщения",
	"тенге", "тг", "руб",
	// kk
	"маман", "кім жасайды", "кім жасап", "жасап беретін", "жекеге", "жекеме", "ақысы", "төлеймін",
	// en
	"someone", "anyone", "somebody", "budget", "pay", "paid", "il pay", "dm me", "quote", "freelancer",
}

// SeekSubstrings are currency signs: plain substring match, they are not words.
var SeekSubstrings = []string{"$", "₸", "₽", "€"}

var NegativePatterns = []string{
	// job seekers
	"ищу работу", "ищу удаленную работу", "ищу удалённую работу", "в поиске работы", "найти работу",
	"ищу оффер", "в поиске оффера", "ищу стажировку", "ищу вакансию", "ищу позицию",
	"ищу удалённую позицию", "ищу удаленную позицию", "ищу удалённую вакансию", "ищу удаленную вакансию",
	"ищу тиммейта", "ищу наставника", "ищу ментора",
	"open to work", "opentowork", "looking for a job", "looking for job",
	"looking for work", "looking for new opportunities", "looking for opportunities",
	"my resume", "моё резюме", "мое резюме", "рассмотрю предложения",
	"available for hire", "available for work", "hire me",
	"looking for a remote role", "looking for remote job", "looking for a role", "looking for remote role",
	"looking for new clients", "looking for clients",
	// sellers advertising their own services, not buyers
	"делаю под ключ", "делаю сайты", "делаю сайт", "создаю сайты", "разрабатываю сайты", "разрабатываю",
	"мои услуги", "наши услуги", "принимаю заказы", "беру заказы", "беру проекты", "открыт к заказам",
	"открыт для заказов", "ищу заказы", "ищу проекты", "ищу клиентов", "помогу с", "обращайтесь",
	"один из моих проектов", "мой проект", "мой кейс", "наш кейс", "показываю, как", "для меня важно",
	"i build", "we build", "i create websites", "i design websites", "i make websites", "we create",
	"our services", "my services", "we offer", "i offer", "open for projects", "available for projects",
	"taking new clients", "accepting new clients", "dm me if you need", "if you need a website",
	"if you need a developer", "need a website? ", "need a developer? ", "нужен сайт? ", "нужен разработчик? ",
	"если вам нужен", "если вам нужна", "если вам нужно", "если нужен сайт", "если нужен разработчик", "if you need",
	// construction / non-IT "engineer"
	"civil engineer", "site engineer", "m&e engineer", "project engineer", "quantity surveyor", "high rise",
	"инженер пто", "инженера пто", "кипиа", "асутп", "инженер-строитель", "инженер по охране труда",
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
