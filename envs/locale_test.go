package envs_test

import (
	"testing"

	"github.com/nyaruka/goflow/envs"
	"github.com/stretchr/testify/assert"
)

func TestLocale(t *testing.T) {
	assert.Equal(t, envs.Locale(""), envs.NewLocale("", ""))
	assert.Equal(t, envs.Locale(""), envs.NewLocale("", "US"))     // invalid without language
	assert.Equal(t, envs.Locale("eng"), envs.NewLocale("eng", "")) // valid without country
	assert.Equal(t, envs.Locale("eng-US"), envs.NewLocale("eng", "US"))

	l, c := envs.Locale("eng-US").ToParts()
	assert.Equal(t, envs.Language("eng"), l)
	assert.Equal(t, envs.Country("US"), c)

	l, c = envs.NilLocale.ToParts()
	assert.Equal(t, envs.NilLanguage, l)
	assert.Equal(t, envs.NilCountry, c)

	v, err := envs.NewLocale("eng", "US").Value()
	assert.NoError(t, err)
	assert.Equal(t, "eng-US", v)

	v, err = envs.NilLanguage.Value()
	assert.NoError(t, err)
	assert.Nil(t, v)

	var lc envs.Locale
	assert.NoError(t, lc.Scan("eng-US"))
	assert.Equal(t, envs.Locale("eng-US"), lc)

	assert.NoError(t, lc.Scan(nil))
	assert.Equal(t, envs.NilLocale, lc)
}

func TestToBCP47(t *testing.T) {
	tests := []struct {
		locale envs.Locale
		bcp47  string
	}{
		{``, ``},
		{`cat`, `ca`},
		{`deu`, `de`},
		{`eng`, `en`},
		{`fin`, `fi`},
		{`fra`, `fr`},
		{`jpn`, `ja`},
		{`kor`, `ko`},
		{`pol`, `pl`},
		{`por`, `pt`},
		{`rus`, `ru`},
		{`spa`, `es`},
		{`swe`, `sv`},
		{`zho`, `zh`},
		{`eng-US`, `en-US`},
		{`spa-EC`, `es-EC`},
		{`zho-CN`, `zh-CN`},

		{`yue`, ``}, // has no 2-letter represention
		{`und`, ``},
		{`mul`, ``},
		{`xyz`, ``}, // is not a language
	}

	for _, tc := range tests {
		assert.Equal(t, tc.bcp47, tc.locale.ToBCP47())
	}
}
