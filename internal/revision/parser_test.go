package revision

import (
	"bytes"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ParserSuite struct {
	suite.Suite
}

func TestParserSuite(t *testing.T) {
	suite.Run(t, new(ParserSuite))
}

func (s *ParserSuite) TestErrInvalidRevision() {
	e := ErrInvalidRevision{"test"}

	s.Equal("Revision invalid : test", e.Error())
}

func (s *ParserSuite) TestNewParserFromString() {
	p := NewParserFromString("test")

	s.IsType(&Parser{}, p)
}

func (s *ParserSuite) TestScan() {
	parser := NewParser(bytes.NewBufferString("Hello world !"))

	expected := []struct {
		t token
		s string
	}{
		{
			word,
			"Hello",
		},
		{
			space,
			" ",
		},
		{
			word,
			"world",
		},
		{
			space,
			" ",
		},
		{
			emark,
			"!",
		},
	}

	for i := 0; ; {
		tok, str, err := parser.scan()

		if tok == eof {
			return
		}

		s.NoError(err)
		s.Equal(expected[i].s, str)
		s.Equal(expected[i].t, tok)

		i++
	}
}

func (s *ParserSuite) TestUnscan() {
	parser := NewParser(bytes.NewBufferString("Hello world !"))

	tok, str, err := parser.scan()

	s.NoError(err)
	s.Equal("Hello", str)
	s.Equal(word, tok)

	parser.unscan()

	tok, str, err = parser.scan()

	s.NoError(err)
	s.Equal("Hello", str)
	s.Equal(word, tok)
}

func (s *ParserSuite) TestParseWithValidExpression() {
	tim, _ := time.Parse("2006-01-02T15:04:05Z", "2016-12-16T21:42:47Z")

	datas := map[string]Revisioner{
		"@": []Revisioner{Ref("HEAD")},
		"@~3": []Revisioner{
			Ref("HEAD"),
			TildePath{3},
		},
		"@{2016-12-16T21:42:47Z}": []Revisioner{AtDate{tim}},
		"@{1}":                    []Revisioner{AtReflog{1}},
		"@{-1}":                   []Revisioner{AtCheckout{1}},
		"master@{upstream}": []Revisioner{
			Ref("master"),
			AtUpstream{},
		},
		"@{upstream}": []Revisioner{
			AtUpstream{},
		},
		"@{u}": []Revisioner{
			AtUpstream{},
		},
		"master@{push}": []Revisioner{
			Ref("master"),
			AtPush{},
		},
		"master@{2016-12-16T21:42:47Z}": []Revisioner{
			Ref("master"),
			AtDate{tim},
		},
		"HEAD^": []Revisioner{
			Ref("HEAD"),
			CaretPath{1},
		},
		"master~3": []Revisioner{
			Ref("master"),
			TildePath{3},
		},
		"v0.99.8^{commit}": []Revisioner{
			Ref("v0.99.8"),
			CaretType{"commit"},
		},
		"v0.99.8^{}": []Revisioner{
			Ref("v0.99.8"),
			CaretType{"tag"},
		},
		"HEAD^{/fix nasty bug}": []Revisioner{
			Ref("HEAD"),
			CaretReg{regexp.MustCompile("fix nasty bug"), false},
		},
		":/fix nasty bug": []Revisioner{
			ColonReg{regexp.MustCompile("fix nasty bug"), false},
		},
		"HEAD:README": []Revisioner{
			Ref("HEAD"),
			ColonPath{"README"},
		},
		":README": []Revisioner{
			ColonPath{"README"},
		},
		"master:./README": []Revisioner{
			Ref("master"),
			ColonPath{"./README"},
		},
		"master^1~:./README": []Revisioner{
			Ref("master"),
			CaretPath{1},
			TildePath{1},
			ColonPath{"./README"},
		},
		":0:README": []Revisioner{
			ColonStagePath{"README", 0},
		},
		":3:README": []Revisioner{
			ColonStagePath{"README", 3},
		},
		"master~1^{/update}~5~^^1": []Revisioner{
			Ref("master"),
			TildePath{1},
			CaretReg{regexp.MustCompile("update"), false},
			TildePath{5},
			TildePath{1},
			CaretPath{1},
			CaretPath{1},
		},
	}

	for d, expected := range datas {
		parser := NewParser(bytes.NewBufferString(d))

		result, err := parser.Parse()

		s.NoError(err)
		s.Equal(expected, result)
	}
}

func (s *ParserSuite) TestParseWithInvalidExpression() {
	datas := map[string]error{
		"..":                              &ErrInvalidRevision{`must not start with "."`},
		"master^1master":                  &ErrInvalidRevision{`reference must be defined once at the beginning`},
		"master^1@{2016-12-16T21:42:47Z}": &ErrInvalidRevision{`"@" statement is not valid, could be : <refname>@{<ISO-8601 date>}, @{<ISO-8601 date>}`},
		"master^1@{1}":                    &ErrInvalidRevision{`"@" statement is not valid, could be : <refname>@{<n>}, @{<n>}`},
		"master@{-1}":                     &ErrInvalidRevision{`"@" statement is not valid, could be : @{-<n>}`},
		"master^1@{upstream}":             &ErrInvalidRevision{`"@" statement is not valid, could be : <refname>@{upstream}, @{upstream}, <refname>@{u}, @{u}`},
		"master^1@{u}":                    &ErrInvalidRevision{`"@" statement is not valid, could be : <refname>@{upstream}, @{upstream}, <refname>@{u}, @{u}`},
		"master^1@{push}":                 &ErrInvalidRevision{`"@" statement is not valid, could be : <refname>@{push}, @{push}`},
		"^1":                              &ErrInvalidRevision{`"~" or "^" statement must have a reference defined at the beginning`},
		"^{/test}":                        &ErrInvalidRevision{`"~" or "^" statement must have a reference defined at the beginning`},
		"~1":                              &ErrInvalidRevision{`"~" or "^" statement must have a reference defined at the beginning`},
		"master:/test":                    &ErrInvalidRevision{`":" statement is not valid, could be : :/<regexp>`},
		"master:0:README":                 &ErrInvalidRevision{`":" statement is not valid, could be : :<n>:<path>`},
		"^{/":                             &ErrInvalidRevision{`missing "}" in ^{<data>} structure`},
		"~@{":                             &ErrInvalidRevision{`missing "}" in @{<data>} structure`},
		"@@{{0":                           &ErrInvalidRevision{`missing "}" in @{<data>} structure`},
	}

	for st, e := range datas {
		parser := NewParser(bytes.NewBufferString(st))
		_, err := parser.Parse()
		s.Equal(err, e)
	}
}

func (s *ParserSuite) TestParseAtWithValidExpression() {
	tim, _ := time.Parse("2006-01-02T15:04:05Z", "2016-12-16T21:42:47Z")

	datas := map[string]Revisioner{
		"":                       Ref("HEAD"),
		"{1}":                    AtReflog{1},
		"{-1}":                   AtCheckout{1},
		"{push}":                 AtPush{},
		"{upstream}":             AtUpstream{},
		"{u}":                    AtUpstream{},
		"{2016-12-16T21:42:47Z}": AtDate{tim},
	}

	for d, expected := range datas {
		parser := NewParser(bytes.NewBufferString(d))

		result, err := parser.parseAt()

		s.NoError(err)
		s.Equal(expected, result)
	}
}

func (s *ParserSuite) TestParseAtWithInvalidExpression() {
	datas := map[string]error{
		"{test}": &ErrInvalidRevision{`wrong date "test" must fit ISO-8601 format : 2006-01-02T15:04:05Z`},
		"{-1":    &ErrInvalidRevision{`missing "}" in @{-n} structure`},
	}

	for st, e := range datas {
		parser := NewParser(bytes.NewBufferString(st))

		_, err := parser.parseAt()

		s.Equal(err, e)
	}
}

func (s *ParserSuite) TestParseCaretWithValidExpression() {
	datas := map[string]Revisioner{
		"":                    CaretPath{1},
		"2":                   CaretPath{2},
		"{}":                  CaretType{"tag"},
		"{commit}":            CaretType{"commit"},
		"{tree}":              CaretType{"tree"},
		"{blob}":              CaretType{"blob"},
		"{tag}":               CaretType{"tag"},
		"{object}":            CaretType{"object"},
		"{/hello world !}":    CaretReg{regexp.MustCompile("hello world !"), false},
		"{/!-hello world !}":  CaretReg{regexp.MustCompile("hello world !"), true},
		"{/!! hello world !}": CaretReg{regexp.MustCompile("! hello world !"), false},
	}

	for d, expected := range datas {
		parser := NewParser(bytes.NewBufferString(d))

		result, err := parser.parseCaret()

		s.NoError(err)
		s.Equal(expected, result)
	}
}

func (s *ParserSuite) TestParseCaretWithUnValidExpression() {
	datas := map[string]error{
		"3":         &ErrInvalidRevision{`"3" found must be 0, 1 or 2 after "^"`},
		"{test}":    &ErrInvalidRevision{`"test" is not a valid revision suffix brace component`},
		"{/!test}":  &ErrInvalidRevision{`revision suffix brace component sequences starting with "/!" others than those defined are reserved`},
		"{/test**}": &ErrInvalidRevision{"revision suffix brace component, error parsing regexp: invalid nested repetition operator: `**`"},
	}

	for st, e := range datas {
		parser := NewParser(bytes.NewBufferString(st))

		_, err := parser.parseCaret()

		s.Equal(err, e)
	}
}

func (s *ParserSuite) TestParseTildeWithValidExpression() {
	datas := map[string]Revisioner{
		"3": TildePath{3},
		"1": TildePath{1},
		"":  TildePath{1},
	}

	for d, expected := range datas {
		parser := NewParser(bytes.NewBufferString(d))

		result, err := parser.parseTilde()

		s.NoError(err)
		s.Equal(expected, result)
	}
}

func (s *ParserSuite) TestParseColonWithValidExpression() {
	datas := map[string]Revisioner{
		"/hello world !":    ColonReg{regexp.MustCompile("hello world !"), false},
		"/!-hello world !":  ColonReg{regexp.MustCompile("hello world !"), true},
		"/!! hello world !": ColonReg{regexp.MustCompile("! hello world !"), false},
		"../parser.go":      ColonPath{"../parser.go"},
		"./parser.go":       ColonPath{"./parser.go"},
		"parser.go":         ColonPath{"parser.go"},
		"0:parser.go":       ColonStagePath{"parser.go", 0},
		"1:parser.go":       ColonStagePath{"parser.go", 1},
		"2:parser.go":       ColonStagePath{"parser.go", 2},
		"3:parser.go":       ColonStagePath{"parser.go", 3},
	}

	for d, expected := range datas {
		parser := NewParser(bytes.NewBufferString(d))

		result, err := parser.parseColon()

		s.NoError(err)
		s.Equal(expected, result)
	}
}

func (s *ParserSuite) TestParseColonWithUnValidExpression() {
	datas := map[string]error{
		"/!test": &ErrInvalidRevision{`revision suffix brace component sequences starting with "/!" others than those defined are reserved`},
		"/*":     &ErrInvalidRevision{"revision suffix brace component, error parsing regexp: missing argument to repetition operator: `*`"},
	}

	for st, e := range datas {
		parser := NewParser(bytes.NewBufferString(st))

		_, err := parser.parseColon()

		s.Equal(err, e)
	}
}

func (s *ParserSuite) TestParseRefWithValidName() {
	datas := []string{
		"lock",
		"master",
		"v1.0.0",
		"refs/stash",
		"refs/tags/v1.0.0",
		"refs/heads/master",
		"refs/remotes/test",
		"refs/remotes/origin/HEAD",
		"refs/remotes/origin/master",
		"0123abcd", // short hash
	}

	for _, d := range datas {
		parser := NewParser(bytes.NewBufferString(d))

		result, err := parser.parseRef()

		s.NoError(err)
		s.Equal(Ref(d), result)
	}
}

func (s *ParserSuite) TestParseRefWithInvalidName() {
	datas := map[string]error{
		".master":                     &ErrInvalidRevision{`must not start with "."`},
		"/master":                     &ErrInvalidRevision{`must not start with "/"`},
		"master/":                     &ErrInvalidRevision{`must not end with "/"`},
		"master.":                     &ErrInvalidRevision{`must not end with "."`},
		"refs/remotes/.origin/HEAD":   &ErrInvalidRevision{`must not contains "/."`},
		"test..test":                  &ErrInvalidRevision{`must not contains ".."`},
		"test..":                      &ErrInvalidRevision{`must not contains ".."`},
		"test test":                   &ErrInvalidRevision{`must not contains " "`},
		"test*test":                   &ErrInvalidRevision{`must not contains "*"`},
		"test?test":                   &ErrInvalidRevision{`must not contains "?"`},
		"test\\test":                  &ErrInvalidRevision{`must not contains "\"`},
		"test[test":                   &ErrInvalidRevision{`must not contains "["`},
		"te//st":                      &ErrInvalidRevision{`must not contains consecutively "/"`},
		"refs/remotes/test.lock/HEAD": &ErrInvalidRevision{`cannot end with .lock`},
		"test.lock":                   &ErrInvalidRevision{`cannot end with .lock`},
	}

	for st, e := range datas {
		parser := NewParser(bytes.NewBufferString(st))

		_, err := parser.parseRef()

		s.Equal(err, e)
	}
}

func FuzzParser(f *testing.F) {
	f.Add("@{2016-12-16T21:42:47Z}")
	f.Add("@~3")
	f.Add("v0.99.8^{}")
	f.Add("master:./README")
	f.Add("HEAD^{/fix nasty bug}")
	f.Add("HEAD^{/[A-")
	f.Add(":/fix nasty bug")
	f.Add(":/[A-")

	f.Fuzz(func(t *testing.T, input string) {
		parser := NewParser(bytes.NewBufferString(input))
		parser.Parse()
	})
}
