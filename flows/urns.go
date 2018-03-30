package flows

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nyaruka/gocommon/urns"
	"github.com/nyaruka/goflow/utils"

	validator "gopkg.in/go-playground/validator.v9"
)

func init() {
	utils.Validator.RegisterValidation("urn", ValidateURN)
	utils.Validator.RegisterValidation("urnscheme", ValidateURNScheme)
}

// ValidateURN validates whether the field value is a valid URN
func ValidateURN(fl validator.FieldLevel) bool {
	err := urns.URN(fl.Field().String()).Validate()
	return err == nil
}

// ValidateURNScheme validates whether the field value is a valid URN scheme
func ValidateURNScheme(fl validator.FieldLevel) bool {
	return urns.IsValidScheme(fl.Field().String())
}

// ContactURN holds a URN for a contact with the channel parsed out
type ContactURN struct {
	urns.URN
	channel Channel
}

// NewContactURN creates a new contact URN with associated channel
func NewContactURN(urn urns.URN, channel Channel) *ContactURN {
	return &ContactURN{URN: urn, channel: channel}
}

// Channel gets the channel associated with this URN
func (u *ContactURN) Channel() Channel { return u.channel }

// SetChannel sets the channel associated with this URN
func (u *ContactURN) SetChannel(channel Channel) { u.channel = channel }

// Resolve resolves the given key when this URN is referenced in an expression
func (u *ContactURN) Resolve(key string) interface{} {
	switch key {
	case "scheme":
		return u.URN.Scheme()
	case "path":
		return u.URN.Path()
	case "display":
		return u.URN.Display()
	case "channel":
		return u.Channel()
	}
	return fmt.Errorf("no field '%s' on URN", key)
}

// Atomize is called when this object needs to be reduced to a primitive
func (u *ContactURN) Atomize() interface{} { return string(u.URN) }

var _ utils.Atomizable = (*ContactURN)(nil)
var _ utils.Resolvable = (*ContactURN)(nil)

// URNList is the list of a contact's URNs
type URNList []*ContactURN

// ReadURNList parses contact URN list from the given list of raw URNs
func ReadURNList(session Session, rawURNs []urns.URN) (URNList, error) {
	l := make(URNList, len(rawURNs))

	for u := range rawURNs {
		scheme, path, query, display := rawURNs[u].ToParts()

		// re-create the URN without the query component
		queryLess, err := urns.NewURNFromParts(scheme, path, "", display)
		if err != nil {
			return nil, err
		}

		parsedQuery, err := url.ParseQuery(query)
		if err != nil {
			return nil, err
		}

		var channel Channel
		channelUUID := parsedQuery.Get("channel")
		if channelUUID != "" {
			if channel, err = session.Assets().GetChannel(ChannelUUID(channelUUID)); err != nil {
				return nil, err
			}
		}

		l[u] = &ContactURN{URN: queryLess, channel: channel}
	}
	return l, nil
}

// RawURNs returns the raw URNs with or without channel information
func (l URNList) RawURNs(includeChannels bool) []urns.URN {
	raw := make([]urns.URN, len(l))
	for u := range l {
		scheme, path, query, display := l[u].URN.ToParts()

		if includeChannels && l[u].channel != nil {
			query = fmt.Sprintf("channel=%s", l[u].channel.UUID())
		}

		raw[u], _ = urns.NewURNFromParts(scheme, path, query, display)
	}
	return raw
}

// Clone returns a clone of this URN list
func (l URNList) clone() URNList {
	urns := make(URNList, len(l))
	copy(urns, l)
	return urns
}

// WithScheme returns a new URN list containing of only URNs of the given scheme
func (l URNList) WithScheme(scheme string) URNList {
	var matching URNList
	for _, u := range l {
		if u.URN.Scheme() == scheme {
			matching = append(matching, u)
		}
	}
	return matching
}

// Resolve resolves the given key when this URN list is referenced in an expression
func (l URNList) Resolve(key string) interface{} {
	scheme := strings.ToLower(key)

	// if this isn't a valid scheme, bail
	if !urns.IsValidScheme(scheme) {
		return fmt.Errorf("unknown URN scheme: %s", key)
	}

	return l.WithScheme(scheme)
}

// Atomize is called when this object needs to be reduced to a primitive
func (l URNList) Atomize() interface{} {
	array := utils.NewArray()
	for _, urn := range l {
		array.Append(urn)
	}
	return array
}

// Index is called when this object is indexed into in an expression
func (l URNList) Index(index int) interface{} {
	return l[index]
}

// Length is called when the length of this object is requested in an expression
func (l URNList) Length() int {
	return len(l)
}

var _ utils.Atomizable = (URNList)(nil)
var _ utils.Indexable = (URNList)(nil)
var _ utils.Resolvable = (URNList)(nil)
