package bothub

import (
	"net/http"
	"strings"

	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/stringsx"
	"github.com/nyaruka/goflow/envs"
	"github.com/nyaruka/goflow/flows"
)

// a classification service implementation for a bothub.it bot
type service struct {
	client     *Client
	classifier *flows.Classifier
	redactor   stringsx.Redactor
}

// NewService creates a new classification service
func NewService(httpClient *http.Client, httpRetries *httpx.RetryConfig, classifier *flows.Classifier, accessToken string) flows.ClassificationService {
	return &service{
		client:     NewClient(httpClient, httpRetries, accessToken),
		classifier: classifier,
		redactor:   stringsx.NewRedactor(flows.RedactionMask, accessToken),
	}
}

func (s *service) Classify(env envs.Environment, input string, logHTTP flows.HTTPLogCallback) (*flows.Classification, error) {
	localeStr := strings.ReplaceAll(strings.ToLower(env.DefaultLocale().ToBCP47()), "-", "_") // en-US -> en_us

	response, trace, err := s.client.Parse(input, localeStr)
	if trace != nil {
		logHTTP(flows.NewHTTPLog(trace, flows.HTTPStatusFromCode, s.redactor))
	}
	if err != nil {
		return nil, err
	}

	result := &flows.Classification{
		Intents:  make([]flows.ExtractedIntent, len(response.IntentRanking)),
		Entities: make(map[string][]flows.ExtractedEntity, len(response.LabelsList)),
	}

	for i, intent := range response.IntentRanking {
		result.Intents[i] = flows.ExtractedIntent{Name: intent.Name, Confidence: intent.Confidence}
	}

	for label, entities := range response.Entities {
		result.Entities[label] = make([]flows.ExtractedEntity, 0, len(response.Entities))

		for _, entity := range entities {
			result.Entities[label] = append(result.Entities[label], flows.ExtractedEntity{Value: entity.Entity, Confidence: entity.Confidence})
		}
	}

	return result, nil
}

var _ flows.ClassificationService = (*service)(nil)
