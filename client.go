package anthropic

type (
	Client struct {
		apiKey  string
		baseUrl string
	}
)

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseUrl: "https://api.anthropic.com/v1",
	}
}
