package mtproto

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
)

type MediaItem struct {
	Index    string `json:"index"`
	MsgID    int    `json:"msg_id"`
	Type     string `json:"type"` // "video" or "pdf"
	Title    string `json:"title"`
	Duration int    `json:"duration,omitempty"`
	Size     int64  `json:"size"`
	ThumbID  string `json:"thumb_id,omitempty"`
}

type BatchData struct {
	ChannelName string                 `json:"channel_name"`
	ChatID      string                 `json:"chat_id"`
	Subjects    map[string][]MediaItem `json:"subjects"`
	TotalVideos int                    `json:"total_videos"`
	TotalPDFs   int                    `json:"total_pdfs"`
	LastUpdated int64                  `json:"last_updated"`
}

// ExtractMetadata implements the Python regex extraction for Index, Title, Subject
func ExtractMetadata(caption string) (index string, title string, subject string) {
	idxRe := regexp.MustCompile(`(?i)Index:\s*(.*)`)
	titleRe := regexp.MustCompile(`(?i)Title:\s*(.*)`)
	subRe := regexp.MustCompile(`(?i)Subject:\s*(.*)`)

	if m := idxRe.FindStringSubmatch(caption); len(m) > 1 {
		index = m[1]
	} else {
		index = "999"
	}

	if m := titleRe.FindStringSubmatch(caption); len(m) > 1 {
		title = m[1]
	} else {
		title = "Unknown Media"
	}

	if m := subRe.FindStringSubmatch(caption); len(m) > 1 {
		subject = m[1]
	} else {
		subject = "Other Files"
	}

	return index, title, subject
}

// ScanBatch implements the equivalent of cmd_storebatch
func (s *Service) ScanBatch(ctx context.Context, chatID int64, progressCallback func(videoCount, pdfCount int, channelName string)) (string, error) {
	state, err := s.store.Load(ctx)
	if err != nil || state.UserbotSession == "" {
		return "", fmt.Errorf("userbot not logged in (no session string)")
	}

	var sessData session.Data
	if err := json.Unmarshal([]byte(state.UserbotSession), &sessData); err != nil {
		return "", fmt.Errorf("failed to parse session string: %v", err)
	}

	sessionStorage := &session.StorageMemory{}
	loader := session.Loader{Storage: sessionStorage}
	if err := loader.Save(ctx, &sessData); err != nil {
		return "", fmt.Errorf("failed to load session string: %v", err)
	}

	client := telegram.NewClient(s.apiID, s.apiHash, telegram.Options{
		SessionStorage: sessionStorage,
	})

	var finalData *BatchData

	err = client.Run(ctx, func(ctx context.Context) error {
		api := client.API()
		sender := message.NewSender(api)

		peer := sender.Resolve(strconv.FormatInt(chatID, 10))
		peerClass, err := peer.AsInputPeer(ctx)
		if err != nil {
			return err
		}

		channelName := fmt.Sprintf("Batch %d", chatID)

		// Attempt to get chat title
		if chats, err := api.MessagesGetChats(ctx, []int64{chatID}); err == nil {
			if len(chats.GetChats()) > 0 {
				if c, ok := chats.GetChats()[0].(*tg.Channel); ok {
					channelName = c.Title
				}
			}
		}

		subjectsDict := make(map[string][]MediaItem)
		videoCount := 0
		pdfCount := 0

		// Pagination logic (equivalent to async for m in get_chat_history)
		offsetID := 0
		for {
			hist, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
				Peer:     peerClass,
				Limit:    100,
				OffsetID: offsetID,
			})
			if err != nil {
				return err
			}

			messages := hist.(*tg.MessagesChannelMessages).GetMessages()
			if len(messages) == 0 {
				break
			}

			for _, m := range messages {
				msg, ok := m.(*tg.Message)
				if !ok {
					continue
				}

				offsetID = msg.ID

				var caption string
				if msg.Message != "" {
					caption = msg.Message
				}
				idx, title, sub := ExtractMetadata(caption)

				if msg.Media != nil {
					switch media := msg.Media.(type) {
					case *tg.MessageMediaDocument:
						doc, ok := media.Document.(*tg.Document)
						if !ok {
							continue
						}

						isVid := false
						isPDF := false
						fileName := ""

						for _, attr := range doc.Attributes {
							if va, ok := attr.(*tg.DocumentAttributeVideo); ok {
								isVid = true
								_ = va // duration available here
							}
							if fa, ok := attr.(*tg.DocumentAttributeFilename); ok {
								fileName = fa.FileName
							}
						}

						if doc.MimeType == "application/pdf" {
							isPDF = true
						}

						if isVid {
							videoCount++
							if title == "Unknown Media" && fileName != "" {
								title = fileName
							}
							subjectsDict[sub] = append(subjectsDict[sub], MediaItem{
								Index: idx,
								MsgID: msg.ID,
								Type:  "video",
								Title: title,
								Size:  doc.Size,
							})
						} else if isPDF {
							pdfCount++
							if title == "Unknown Media" && fileName != "" {
								title = fileName
							}
							subjectsDict[sub] = append(subjectsDict[sub], MediaItem{
								Index: idx,
								MsgID: msg.ID,
								Type:  "pdf",
								Title: title,
								Size:  doc.Size,
							})
						}
					}
				}
			}

			// Callback for UI updates
			if progressCallback != nil && (videoCount+pdfCount)%300 == 0 {
				progressCallback(videoCount, pdfCount, channelName)
			}
		}

		// Sort by index
		for k, v := range subjectsDict {
			sort.Slice(v, func(i, j int) bool {
				return v[i].Index < v[j].Index
			})
			subjectsDict[k] = v
		}

		finalData = &BatchData{
			ChannelName: channelName,
			ChatID:      strconv.FormatInt(chatID, 10),
			Subjects:    subjectsDict,
			TotalVideos: videoCount,
			TotalPDFs:   pdfCount,
			LastUpdated: time.Now().Unix(),
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	// Mocking Firestore save because Firebase config isn't implemented in Go env yet.
	log.Printf("Simulating Firestore Save: %d videos, %d PDFs scanned for %s", finalData.TotalVideos, finalData.TotalPDFs, finalData.ChannelName)
	return fmt.Sprintf("✅ Scan Complete! Videos: %d, PDFs: %d", finalData.TotalVideos, finalData.TotalPDFs), nil
}
