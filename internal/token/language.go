package token

import (
	"regexp"
	"strings"
)

var wordSeparatorRegex = regexp.MustCompile("[\\s!\"&,./?|]+")

var languagesByName = map[string]string{
	// GoodRx Languages
	// Go
	"go":     "Go",
	"golang": "Go",
	// Kotlin
	"kotlin": "Kotlin",
	"kt":     "Kotlin",
	// Python
	"py":     "Python",
	"python": "Python",
	// Swift
	"swift": "Swift",
	// TypeScript
	"ts":         "TypeScript",
	"typescript": "TypeScript",

	// Others
	// C/C++
	"c":   "C",
	"c++": "C",
	// C#
	"c#":     "C#",
	"csharp": "C#",
	// Java
	"java": "Java",
	// Javascript
	"js":         "JavaScript",
	"ecmascript": "JavaScript",
	"javascript": "JavaScript",
	// Lisp
	"lisp":    "Lisp",
	"clojure": "Lisp",
	"racket":  "Lisp",
	"scheme":  "Lisp",
	// ML
	"ml":         "ML",
	"haskell":    "ML",
	"caml":       "ML",
	"elm":        "ML",
	"f#":         "ML",
	"ocaml":      "ML",
	"purescript": "ML",
	// Perl
	"perl": "Perl",
	// PHP
	"php": "PHP",
	// Ruby
	"ruby": "Ruby",
	"rb":   "Ruby",
	// Rust
	"rust": "Rust",
	// Scala
	"scala": "Scala",
}

func ExtractLanguages(text string) []string {
	words := wordSeparatorRegex.Split(text, -1)
	languages := make([]string, 0, len(words))
	for _, word := range words {
		language := languagesByName[strings.ToLower(word)]

		if language != "" {
			languages = append(languages, language)
		}
	}

	return languages
}
