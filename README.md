# go-anthropic

```
package main

import (
	"fmt"
    "log"
    "io"

    "github.com/kznrluk/go-anthropic"
)

func main() {
	ac := anthropic.NewClient("YOUR_API_KEY")

	stream, err := ac.CreateMessageStream(anthropic.MessageRequest{
		MaxTokens: 4096,
		Model:     anthropic.Claude3Opus20240229,
		System:    "System message",
		Messages:  []anthropic.Message{
            {
                Role: "user",
                Text: "Why are you so good at generating text?",
            },
        }
	})
	
	data := ""
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Fatalf("stream.Recv() failed: %v", err)
            }
		}

		fmt.Printf("%s", resp.Delta.Text)
		data += resp.Delta.Text
	}

}
```