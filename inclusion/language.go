package inclusion

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type InclusiveFilter struct {
	Filter string // Supports regex
	Reply  string
	regex  *regexp.Regexp // do not fill. Just used for caching the regex once compiled.
}

var conductLinks = "\nIn case of doubts please check our <https://bcneng.org/coc|Code of Conduct> and/or our <https://bcneng.org/netiquette|Netiquette> "

var inclusiveFilters = []InclusiveFilter{
	// When someone says, the bot replies (privately).
	// English: Based on https://github.com/randsleadershipslack/documents-and-resources/blob/master/RandsInclusiveLanguage.tsv
	{Filter: "you guys", Reply: "Instead of *guys*, perhaps you mean *pals*? You can read more information about it at https://www.dictionary.com/e/you-guys/... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "these guys", Reply: "Instead of *guys*, perhaps you mean *gang*? You can read more information about it at https://www.dictionary.com/e/you-guys/... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "my guys", Reply: "Instead of *guys*, perhaps you mean *crew*? You can read more information about it at https://www.dictionary.com/e/you-guys/... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "those guys", Reply: "Instead of *guys*, perhaps you mean *people*? You can read more information about it at https://www.dictionary.com/e/you-guys/... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "hey guys", Reply: "Instead of *guys*, perhaps you mean *y'all*? You can read more information about it at https://www.dictionary.com/e/you-guys/... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "hi guys", Reply: "Instead of *guys*, perhaps you mean *everyone*? You can read more information about it at https://www.dictionary.com/e/you-guys/... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "the guys", Reply: "Instead of *guys*, perhaps you mean *folks*? You can read more information about it at https://www.dictionary.com/e/you-guys/... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "guys", Reply: "Instead of *guys*, have you considered a more gender-neutral pronoun like *folks*? You can read more information about it at https://www.dictionary.com/e/you-guys/... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "CHWD", Reply: `Cisgender Hetero White Dude. But please consider using the full term "cisgender, heterosexual white man” or similar. That would both make it more approachable for those unfamiliar with this obscure initialism, and prevent reducing people down to initialisms.`},
	{Filter: "URP", Reply: `Underrepresented person(s). But please consider using the full term "members of traditionally underrepresented groups" or similar; people don't like to be made into acronyms, _especially_ when they are already marginalized. See: en.wikipedia.org/wiki/Underrepresented_group`},
	{Filter: "URPs", Reply: `Underrepresented person(s). But please consider using the full term "members of traditionally underrepresented groups" or similar; people don't like to be made into acronyms, _especially_ when they are already marginalized. See: en.wikipedia.org/wiki/Underrepresented_group`},
	{Filter: "URM", Reply: `Underrepresented minorit(y|ies). But please consider using the full term "members of traditionally underrepresented groups" or similar; people don't like to be made into acronyms, _especially_ when they are already marginalized. See: en.wikipedia.org/wiki/Underrepresented_group`},
	{Filter: "URG", Reply: `Underrepresented group(s). But please consider using the full term "members of traditionally underrepresented groups" or similar; people don't like to be made into acronyms, _especially_ when they are already marginalized. See: en.wikipedia.org/wiki/Underrepresented_group`},
	{Filter: "crazy", Reply: "Using the word *crazy* is considered by some to be insensitive to sufferers of mental illness, maybe you mean *outrageous*, *unthinkable*, *nonsensical*, *incomprehensible*? Have you considered a different adjective like *ridiculous*? You can read more information about it at https://www.selfdefined.app/definitions/crazy/"},
	{Filter: "insane", Reply: "The word *insane* is considered by some to be insensitive to sufferers of mental illness. Perhaps you mean *outrageous*, *unthinkable*, *nonsensical*, *incomprehensible*? Have you considered a different adjective like *ridiculous*? You can read more information about it at https://www.selfdefined.app/definitions/crazy/"},
	{Filter: "slave", Reply: `If you are referring to a data replication strategy, please consider a term such as ""follower"" or ""replica"". You can read more information about it at https://www.selfdefined.app/definitions/master-slave/`},

	// Spanish: Based on https://www.cocemfe.es/wp-content/uploads/2019/02/20181010_COCEMFE_Lenguaje_inclusivo.pdf
	{Filter: "discapacitad(a|o)", Reply: "Ante todo somos personas, y no queremos que se nos etiquete, puesto que la discapacidad es una característica más de todas las que se tiene, no lo único por lo que se debe reconocer.\nPor eso es importante anteponer la palabra *persona* y lo más aconsejable es utilizar el término *persona con discapacidad* y no *discapacitado*.\nMás info en https://www.cocemfe.es/wp-content/uploads/2019/02/20181010_COCEMFE_Lenguaje_inclusivo.pdf"},
	{Filter: "discapacitad(a|o) fisic(a|o)", Reply: "Ante todo somos personas, y no queremos que se nos etiquete, puesto que la discapacidad es una característica más de todas las que se tiene, no lo único por lo que se debe reconocer.\nPor eso es importante anteponer la palabra *persona* y lo más aconsejable es utilizar el término *persona con discapacidad* y no *discapacitada física*.\nMás info en https://www.cocemfe.es/wp-content/uploads/2019/02/20181010_COCEMFE_Lenguaje_inclusivo.pdf"},
	{Filter: "minusvalid(a|o)", Reply: "*Minusválido* es un término peyorativo y vulnera la dignidad de las personas con discapacidad, al atribuirse un nulo o reducido valor a una persona, o utilizarse generalmente con elevada carga negativa. Considera usar *persona con discapacidad*.\nMás info en Más info en https://www.cocemfe.es/wp-content/uploads/2019/02/20181010_COCEMFE_Lenguaje_inclusivo.pdf"},
	{Filter: "diversidad funcional", Reply: "COCEMFE considera que el término *diversidad funcional* es un eufemismo, cargado de condescendencia que genera confusión, inseguridad jurídica y rebaja la protección que todavía es necesaria. El término *discapacidad* es el que aglutina derechos reconocidos legalmente y que cuenta con el mayor respaldo social. Considera usarlo.\nMás info en Más info en https://www.cocemfe.es/wp-content/uploads/2019/02/20181010_COCEMFE_Lenguaje_inclusivo.pdf"},
	{Filter: "retrasad(a|o)", Reply: "*Retrasado* y *Retraso mental* son términos despectivos eliminados del vocabulario psiquiátrico y, según la OMS, la forma correcta para referiste a ese grupo de enfermedades y transtornos es *Trastorno del desarrollo intelectual* "},
	{Filter: "retraso mental", Reply: "*Retrasado* y *Retraso mental* son términos despectivos eliminados del vocabulario psiquiátrico y, según la OMS, la forma correcta para referiste a ese grupo de enfermedades y transtornos es *Trastorno del desarrollo intelectual* "},

	// Our own list
	{Filter: "gentlem(a|e)n", Reply: "Instead of *gentlem(a|e)n*, perhaps you mean *folks*?... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "lad(y|ies)", Reply: "Instead of *lad(y|ies)*, perhaps you mean *folks*?... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "los chicos de", Reply: "En vez de *los chicos de*, quizá quisiste decir *el equipo de*, *los integrantes de*? Para más información (en inglés), puedes consultar https://www.dictionary.com/e/you-guys/... *[Considera editar tu mensaje para que sea más inclusivo]*"},
	{Filter: "chicos", Reply: "En vez de *chicos*, quizá quisiste decir *chiques*, *colegas*, *grupo*, *personas*? Para más información (en inglés), puedes consultar https://www.dictionary.com/e/you-guys/... *[Considera editar tu mensaje para que sea más inclusivo]*"},
	{Filter: "lgtb", Reply: "Desde hace un tiempo, el colectivo *LGTB+* recomienda añadir el carácter `+` a la palabra *LGTB*, pues existen orientaciones e identidades que, a pesar de no ser tan predominantes, representan a muchas personas. *[Considera editar tu mensaje para que sea más inclusivo]*"},
	{Filter: "locura", Reply: "La palabra *locura* es considerada por algunas personas como irrespetuosa hacia las personas que sufren alguna enfermedad mental.\nQuizá quisiste decir *indignante*, *impensable*, *absurdo*, *incomprensible*? Has considerado usar un adjetivo diferente como *ridículo*?"},
	{Filter: "locuron", Reply: "La palabra *locurón* o *locura* es considerada por algunas personas como irrespetuosa hacia las personas que sufren alguna enfermedad mental.\nQuizá quisiste decir *indignante*, *impensable*, *absurdo*, *incomprensible*? Has considerado usar un adjetivo diferente como *ridículo*?"},
	{Filter: "loc(a|o)", Reply: "La palabra *loco/loca* es considerada por algunas personas como irrespetuosa hacia las personas que sufren alguna enfermedad mental.\nQuizá quisiste decir *indignante*, *impensable*, *absurdo*, *incomprensible*? Has considerado usar un adjetivo diferente como *ridículo*?"},
	{Filter: "cakewalk", Reply: "Instead of *cakewalk*, perhaps you mean *easy*?... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "grandfathered in", Reply: "Instead of *grandfathered in*, perhaps you mean *exempting*? You can read more information in https://www.selfdefined.app/definitions/grandfathering/ ... *[Please consider editing your message so it's more inclusive]*"},
	{Filter: "grandfathering", Reply: "Instead of *grandfathering*, perhaps you mean *exempting*? You can read more information in https://www.selfdefined.app/definitions/grandfathering/ ... *[Please consider editing your message so it's more inclusive]*"},
}

type FilteredText struct {
	StopWord string
	Reply    string
}

func Filter(input string, extraFilters ...InclusiveFilter) *InclusiveFilter {
	// Removing accents and others before matching
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	text, _, _ := transform.String(t, strings.ToLower(input))

	filters := append(inclusiveFilters, extraFilters...)
	for _, word := range filters {
		if word.regex == nil {
			// If it's just one word, ensure its bounded as it should.
			if !strings.Contains(word.Filter, " ") {
				word.Filter = fmt.Sprintf("(?:^|\\W)%s(?:$|[^\\w+])", word.Filter)
			}

			word.regex, _ = regexp.Compile(word.Filter)
		}

		if word.regex.MatchString(text) {
			word.Reply += conductLinks
			return &word
		}
	}

	return nil
}
