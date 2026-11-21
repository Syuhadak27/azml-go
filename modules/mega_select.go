package modules

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode"

	"azml-go/config"
	"azml-go/helper/downloader"
	"azml-go/helper/ext"
	"azml-go/helper/telegram"
	"azml-go/web"

	"github.com/gotd/td/tg"
)

type MegaSelectCacheItem struct {
	Files       []downloader.MegaFileInfo
	DoneChan    chan bool
	IsCancelled bool
	UserID      int64
}

var (
	megaSelectCache sync.Map
)

func BuildMegaSelectionButtons(uid string) tg.ReplyMarkupClass {
	cfg := config.Get()
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.BaseURLPort)
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	pinCode := ""
	for _, ch := range uid {
		if unicode.IsDigit(ch) {
			pinCode += string(ch)
			if len(pinCode) == 4 {
				break
			}
		}
	}
	if len(pinCode) < 4 {
		pinCode = "0000"
	}

	b := telegram.NewButtonMaker()
	b.UButtonStyled("◈ "+telegram.ToSmallCaps("Pilih Berkas"), fmt.Sprintf("%s/app/files/%s", baseURL, uid), "primary")
	b.IButtonStyled("◈ "+telegram.ToSmallCaps("Kode PIN"), fmt.Sprintf("megasel pin %s %s", uid, pinCode), "primary")
	b.IButtonStyled("× "+telegram.ToSmallCaps("Batalkan"), fmt.Sprintf("megasel cancel %s", uid), "danger")
	b.IButtonStyled("✓ "+telegram.ToSmallCaps("Selesai Memilih"), fmt.Sprintf("megasel done %s", uid), "success")
	return b.BuildMenu(2)
}

func RegisterMegaSelectCache(uid string, files []downloader.MegaFileInfo, userID int64) chan bool {
	doneChan := make(chan bool, 1)
	megaSelectCache.Store(uid, &MegaSelectCacheItem{
		Files:    files,
		DoneChan: doneChan,
		UserID:   userID,
	})
	web.MegaStore.SaveData(uid, files)
	return doneChan
}

func GetMegaSelectCache(uid string) *MegaSelectCacheItem {
	if val, ok := megaSelectCache.Load(uid); ok {
		return val.(*MegaSelectCacheItem)
	}
	return nil
}

func ClearMegaSelectCache(uid string) {
	megaSelectCache.Delete(uid)
	web.MegaStore.DeleteData(uid)
}

func HandleMegaSelectCallback(ctx context.Context, rawAPI *tg.Client, sender *telegram.MessageSender, peer tg.InputPeerClass, queryID int64, msgID int, data string, fromUID int64) {
	parts := strings.Fields(data)
	if len(parts) < 3 {
		return
	}
	action := parts[1]
	uid := parts[2]

	cache := GetMegaSelectCache(uid)
	if cache == nil {
		_ = sender.AnswerCallback(ctx, queryID, "⌬ Sesi kedaluwarsa atau unduhan sudah dimulai!", true)
		_ = sender.DeleteMessage(ctx, peer, []int{msgID})
		return
	}

	if fromUID != 0 && fromUID != cache.UserID {
		_ = sender.AnswerCallback(ctx, queryID, "× Pilihan ini bukan untuk Anda!", true)
		return
	}

	switch action {
	case "pin":
		pincode := ""
		if len(parts) > 3 {
			pincode = parts[3]
		}
		_ = sender.AnswerCallback(ctx, queryID, fmt.Sprintf("⌬ PIN: %s", pincode), true)

	case "done":
		selectedIDs := web.MegaStore.ReadSelection(uid)
		var selected []downloader.MegaFileInfo
		for i := range cache.Files {
			cache.Files[i].Selected = containsString(selectedIDs, cache.Files[i].ID)
			if cache.Files[i].Selected && !cache.Files[i].IsDir {
				selected = append(selected, cache.Files[i])
			}
		}

		if len(selected) == 0 {
			_ = sender.AnswerCallback(ctx, queryID, "⌬ Pilih setidaknya satu berkas terlebih dahulu!", true)
			return
		}

		var totalSize int64
		for _, f := range selected {
			totalSize += f.Size
		}

		_ = sender.AnswerCallback(ctx, queryID, fmt.Sprintf("⌬ Memulai unduhan %d berkas (%s)!", len(selected), ext.GetReadableFileSize(totalSize)), false)
		_ = sender.DeleteMessage(ctx, peer, []int{msgID})

		select {
		case cache.DoneChan <- true:
		default:
		}

	case "cancel":
		cache.IsCancelled = true
		_ = sender.AnswerCallback(ctx, queryID, "× Unduhan dibatalkan!", false)
		_ = sender.DeleteMessage(ctx, peer, []int{msgID})
		select {
		case cache.DoneChan <- false:
		default:
		}
		ClearMegaSelectCache(uid)
	}
}

func containsString(arr []string, val string) bool {
	for _, a := range arr {
		if a == val {
			return true
		}
	}
	return false
}
