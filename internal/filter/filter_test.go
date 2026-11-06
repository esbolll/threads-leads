package filter

import (
	"testing"
	"time"

	"github.com/esbolll/threads-leads/internal/lead/model"
)

func TestClassify(t *testing.T) {
	// Rule tests are language-agnostic; the region filter has its own test.
	RegionFilter = false
	defer func() { RegionFilter = true }()
	cases := []struct {
		text string
		want bool
		cat  string
	}{
		{"Ищу DevOps для настройки Kubernetes и CI/CD.", true, "unknown"},
		{"Looking for a DevOps engineer for a 2 month AWS migration project.", true, "project"},
		{"Нужен Python разработчик на небольшой проект.", true, "project"},
		{"Сегодня изучал Kubernetes.", false, ""},
		{"DevOps is dead.", false, ""},
		{"5 лучших инструментов для разработчика.", false, ""},
		{"Ищу работу devops, рассмотрю предложения", false, ""},
		{"I need a needle, not a developer", true, "unknown"},
		// real posts from the first runs
		{"I need help opening a website il pay $2500 to anyone who can help me", true, "unknown"},
		{"Hi we are looking for developers to build an event management website for $2500 with admin feature.", true, "unknown"},
		{"WE ARE HIRING\nAnybody with a Devops experience should check this JD and forward cv to me", true, "job"},
		{"I need someone to create a business website for me .", true, "unknown"},
		{"Senior python developer needed for a short contract", true, "contract"},
		// noise: intent and tech far apart or in the wrong order
		{"Check in Atomy website. There are many things that you need not just only skincare.", false, ""},
		// sellers advertising themselves
		{"Привет! Я Дима, дизайнер и разработчик сайтов. Делаю под ключ — от структуры до дизайна. Нужен сайт? Пишите", false, ""},
		{"Разрабатываю сайты и веб-приложения, создаю AI-агентов. Ищу проекты.", false, ""},
		{"We build websites and apps for startups. Need a website? DM us", false, ""},
	}
	now := time.Now()
	for _, c := range cases {
		lead, ok := Classify(model.Post{Text: c.text}, now)
		if ok != c.want {
			t.Errorf("%q: relevant=%v want %v (tech=%v intent=%v)", c.text, ok, c.want, lead.MatchedTech, lead.MatchedIntent)
			continue
		}
		if ok && lead.Category != c.cat {
			t.Errorf("%q: category=%s want %s", c.text, lead.Category, c.cat)
		}
	}
}

func TestWordMatch(t *testing.T) {
	if wordMatch("needle", "need") {
		t.Error("need must not match inside needle")
	}
	if !wordMatch("ищем devops-инженера", "devops") {
		t.Error("hyphen must act as a word boundary")
	}
}

func TestPostKey(t *testing.T) {
	a := model.Post{Permalink: "https://www.threads.com/@u/post/abc?x=1"}
	b := model.Post{Permalink: "https://www.threads.com/@u/post/abc/"}
	if a.Key() != b.Key() {
		t.Errorf("keys differ: %s vs %s", a.Key(), b.Key())
	}
	c := model.Post{Username: "u", Text: "hi"}
	if c.Key() == "" || c.Key()[:5] != "hash:" {
		t.Errorf("hash key expected, got %s", c.Key())
	}
}

func TestRegionFilter(t *testing.T) {
	RegionFilter = true
	cases := []struct {
		user, text string
		lang       string
		want       bool
	}{
		// Russian and Kazakh always pass
		{"someone", "Нужен Python разработчик на небольшой проект.", LangRU, true},
		{"someone", "Ищу DevOps для настройки Kubernetes и CI/CD.", LangRU, true},
		{"someone", "Сайт жасап беретін бағдарламашы керек, бюджет бар", LangKK, true},
		{"someone", "Телеграм бот жасау керек, кім жасайды?", LangKK, true},
		// English without a regional marker is dropped
		{"berlin_ceo070", "I need website developers for restaurant budget 350$", LangEN, false},
		{"xythonix", "Hiring a DevOps Engineer (AWS + CI/CD + Kubernetes)", LangEN, false},
		// English with a regional marker passes
		{"someone", "Looking for a backend developer in Almaty, office or remote", LangEN, true},
		{"himark.kz", "Need a developer for a landing page", LangEN, true},
		{"someone", "Hiring frontend developer, Kazakhstan 🇰🇿, salary in KZT", LangEN, true},
		{"someone", "Need a devops engineer, contact +7 701 123 45 67", LangEN, true},
		// real noise from the first API pass: everyday talk around "сайт", "приложение", "заказ"
		{"someone", "Большой сайт нужен только очень большой организации. Остальным - стильный крутой лендинг", LangRU, false},
		{"someone", "Со временем главный экран телевизора превращается в рекламную площадку, где найти нужное приложение сложнее. Проект", LangRU, false},
		{"someone", "Я бы не искала одну лучшую CRM - всё зависит от процессов бизнеса. Нужна автоматизация или нет", LangRU, false},
		{"someone", "Один из моих проектов — интернет-магазин премиального бренда. Нужен был сайт", LangRU, false},
		{"someone", "NEW COLLECTION пальто цвет: черный. Заказ через app", LangRU, false},
		// real leads from the same pass
		{"jurek_syru", "‼️ Срочно ‼️ сайт жасайтын маман керек жұмысына ұқыпты тиянақты, жекеме жазыңыз", LangKK, true},
		{"someone", "Привет! Мне нужен лендинг, кстати!!! На днях думала заняться поисками специалиста", LangRU, true},
		{"someone", "Нужен сайт для кофейни, бюджет 300 000 тг, кто сделает?", LangRU, true},
		// Ukrainian is not our market
		{"someone", "Шукаю розробника для проєкту, потрібен сайт під ключ, є бюджет", LangOther, false},
		{"eugeeenia.p", "DevOps інженерка. Шукаю роботу", LangOther, false},
	}
	now := time.Now()
	for _, c := range cases {
		if got := DetectLang(c.text); got != c.lang {
			t.Errorf("%q: lang=%s want %s", c.text, got, c.lang)
		}
		_, ok := Classify(model.Post{Username: c.user, Text: c.text}, now)
		if ok != c.want {
			t.Errorf("%q (@%s): relevant=%v want %v", c.text, c.user, ok, c.want)
		}
	}
}
