package modules

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"azml-go/config"
	"azml-go/helper/ext"
	"azml-go/helper/telegram"
	"azml-go/types"

	tdDownloader "github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
)

type SettingsModule struct {
	rawAPI  *tg.Client
	sender  *telegram.MessageSender
	userMap *types.SafeUserMap
}

func NewSettingsModule(rawAPI *tg.Client, sender *telegram.MessageSender, userMap *types.SafeUserMap) *SettingsModule {
	return &SettingsModule{
		rawAPI:  rawAPI,
		sender:  sender,
		userMap: userMap,
	}
}

func (m *SettingsModule) HandleSettings(ctx context.Context, peer tg.InputPeerClass, msg *types.MessageContext) {
	u := m.userMap.GetOrCreate(msg.UserID)
	parts := strings.Fields(msg.Text)

	// Check if this was a thumbnail command (/setthumb, /t, /thumb, or /us thumb)
	isThumbCmd := false
	if len(parts) >= 1 {
		cmd := strings.ToLower(parts[0])
		if strings.Contains(cmd, "setthumb") || strings.Contains(cmd, "thumb") || cmd == "/t" || strings.HasPrefix(cmd, "/t@") {
			isThumbCmd = true
		}
	}
	if len(parts) >= 2 {
		subCmd := strings.ToLower(parts[1])
		if subCmd == "thumb" || subCmd == "setthumb" || subCmd == "t" {
			isThumbCmd = true
		}
	}

	if isThumbCmd {
		if msg.IsReply && msg.ReplyToID != 0 && m.rawAPI != nil {
			res, err := m.rawAPI.MessagesGetMessages(ctx, []tg.InputMessageClass{
				&tg.InputMessageID{ID: msg.ReplyToID},
			})
			var targetMsg *tg.Message
			if err == nil {
				switch mRes := res.(type) {
				case *tg.MessagesMessages:
					if len(mRes.Messages) > 0 {
						targetMsg, _ = mRes.Messages[0].(*tg.Message)
					}
				case *tg.MessagesMessagesSlice:
					if len(mRes.Messages) > 0 {
						targetMsg, _ = mRes.Messages[0].(*tg.Message)
					}
				case *tg.MessagesChannelMessages:
					if len(mRes.Messages) > 0 {
						targetMsg, _ = mRes.Messages[0].(*tg.Message)
					}
				}
			}
			if targetMsg != nil && targetMsg.Media != nil {
				if photoMedia, ok := targetMsg.Media.(*tg.MessageMediaPhoto); ok {
					if photo, ok := photoMedia.Photo.(*tg.Photo); ok {
						_ = os.MkdirAll("Thumbnails", 0755)
						destThumb := fmt.Sprintf("Thumbnails/%d.jpg", msg.UserID)
						thumbSize := "m"
						if len(photo.Sizes) > 0 {
							switch s := photo.Sizes[len(photo.Sizes)-1].(type) {
							case *tg.PhotoSize:
								thumbSize = s.Type
							case *tg.PhotoSizeProgressive:
								thumbSize = s.Type
							}
						}
						dl := tdDownloader.NewDownloader()
						loc := photo.AsInputPhotoFileLocation(thumbSize)
						if loc != nil {
							_, err := dl.Download(m.rawAPI, loc).ToPath(ctx, destThumb)
							if err == nil {
								u.CustomThumb = destThumb
								_ = ext.DB.SaveUser(u)
								ext.ClearPendingThumbSession(msg.UserID)
								_, _ = m.sender.SendMessage(ctx, peer, "✓ <b>Thumbnail kustom disimpan dari foto yang dibalas!</b> Thumbnail ini akan dilampirkan pada semua berkas leech Anda berikutnya.", msg.MessageID, nil)
								return
							}
						}
					}
				} else if docMedia, ok := targetMsg.Media.(*tg.MessageMediaDocument); ok {
					if doc, ok := docMedia.Document.(*tg.Document); ok {
						_ = os.MkdirAll("Thumbnails", 0755)
						destThumb := fmt.Sprintf("Thumbnails/%d.jpg", msg.UserID)
						dl := tdDownloader.NewDownloader()
						loc := doc.AsInputDocumentFileLocation("")
						if loc != nil {
							_, err := dl.Download(m.rawAPI, loc).ToPath(ctx, destThumb)
							if err == nil {
								u.CustomThumb = destThumb
								_ = ext.DB.SaveUser(u)
								ext.ClearPendingThumbSession(msg.UserID)
								_, _ = m.sender.SendMessage(ctx, peer, "✓ <b>Thumbnail kustom disimpan dari dokumen yang dibalas!</b> Thumbnail ini akan dilampirkan pada semua berkas leech Anda berikutnya.", msg.MessageID, nil)
								return
							}
						}
					}
				}
			}
		}

		// Activate 60s listener
		ext.SetPendingThumbSession(msg.UserID, msg.ChatID, msg.MessageID, 60*time.Second)
		promptText := "⌬ <b><u>Kirim Thumbnail Kustom</u></b>\n\n" +
			"Silakan kirim <b>foto</b> atau dokumen gambar dalam waktu <b>60 detik</b>.\n\n" +
			"<i>Setelah diterima, berkas akan disimpan otomatis sebagai thumbnail kustom Anda.</i>\n" +
			"<i>Kirim /cancel untuk membatalkan.</i>"
		_, _ = m.sender.SendMessage(ctx, peer, promptText, msg.MessageID, nil)
		return
	}

	// Direct command arguments e.g. /us lprefix <text> or /us doc
	if len(parts) >= 2 {
		subCmd := strings.ToLower(parts[1])
		switch subCmd {
		case "doc", "asdoc":
			u.AsDoc = !u.AsDoc
			_ = ext.DB.SaveUser(u)
			st := "Nonaktif"
			if u.AsDoc {
				st = "Aktif"
			}
			_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("◈ <b>Unggah Sebagai Dokumen</b> sekarang: <b>%s</b>", st), msg.MessageID, nil)
			return
		case "autorename":
			if len(parts) >= 3 {
				u.RemnameAuto = strings.Join(parts[2:], " ")
				_ = ext.DB.SaveUser(u)
				_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("✓ <b>Templat AutoRename</b> diatur ke: <code>%s</code>", u.RemnameAuto), msg.MessageID, nil)
				return
			}
			u.Autorename = !u.Autorename
			_ = ext.DB.SaveUser(u)
			st := "Nonaktif"
			if u.Autorename {
				st = "Aktif"
			}
			_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("◈ <b>AutoRename</b> sekarang: <b>%s</b>", st), msg.MessageID, nil)
			return
		case "autothumb", "auto_thumb":
			u.AutoThumb = !u.AutoThumb
			_ = ext.DB.SaveUser(u)
			st := "Nonaktif"
			if u.AutoThumb {
				st = "Aktif"
			}
			_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("◈ <b>Thumbnail Otomatis</b> sekarang: <b>%s</b>", st), msg.MessageID, nil)
			return
		case "method", "rename_method":
			if u.RenameMethod == "auto" {
				u.RenameMethod = "regex"
			} else {
				u.RenameMethod = "auto"
			}
			_ = ext.DB.SaveUser(u)
			_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("• <b>Metode Ganti Nama</b> sekarang: <b>%s</b>", strings.Title(u.RenameMethod)), msg.MessageID, nil)
			return
		case "equalsplits", "esplits":
			u.EqualSplits = !u.EqualSplits
			_ = ext.DB.SaveUser(u)
			st := "Nonaktif"
			if u.EqualSplits {
				st = "Aktif"
			}
			_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("◈ <b>Pemisahan Sama Rata</b> sekarang: <b>%s</b>", st), msg.MessageID, nil)
			return
		case "mediagroup", "mgroup":
			u.MediaGroup = !u.MediaGroup
			_ = ext.DB.SaveUser(u)
			st := "Nonaktif"
			if u.MediaGroup {
				st = "Aktif"
			}
			_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("◈ <b>Grup Media</b> sekarang: <b>%s</b>", st), msg.MessageID, nil)
			return
		case "prefix", "lprefix":
			if len(parts) >= 3 {
				val := strings.Join(parts[2:], " ")
				if strings.EqualFold(val, "clear") || strings.EqualFold(val, "none") {
					u.Prefix = ""
					val = "Dihapus"
				} else {
					u.Prefix = val
				}
				_ = ext.DB.SaveUser(u)
				_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("✓ <b>Awalan Leech</b> diatur ke: <code>%s</code>", val), msg.MessageID, nil)
				return
			}
		case "suffix", "lsuffix":
			if len(parts) >= 3 {
				val := strings.Join(parts[2:], " ")
				if strings.EqualFold(val, "clear") || strings.EqualFold(val, "none") {
					u.Suffix = ""
					val = "Dihapus"
				} else {
					u.Suffix = val
				}
				_ = ext.DB.SaveUser(u)
				_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("✓ <b>Akhiran Leech</b> diatur ke: <code>%s</code>", val), msg.MessageID, nil)
				return
			}
		case "caption", "lcaption":
			if len(parts) >= 3 {
				val := strings.Join(parts[2:], " ")
				if strings.EqualFold(val, "clear") || strings.EqualFold(val, "default") {
					u.Caption = ""
					val = "Bawaan"
				} else {
					u.Caption = val
				}
				_ = ext.DB.SaveUser(u)
				_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("✓ <b>Keterangan Leech</b> diatur ke: <code>%s</code>", val), msg.MessageID, nil)
				return
			}
		case "dump", "ldump":
			if len(parts) >= 3 {
				id, _ := strconv.ParseInt(parts[2], 10, 64)
				u.DumpChatID = id
				_ = ext.DB.SaveUser(u)
				_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("✓ <b>ID Kanal Dump Leech</b> diatur ke: <code>%d</code>", u.DumpChatID), msg.MessageID, nil)
				return
			}
		case "meta", "lmeta", "metadata":
			if len(parts) >= 3 {
				if u.Metadata == nil {
					u.Metadata = make(map[string]string)
				}
				val := strings.Join(parts[2:], " ")
				if strings.EqualFold(val, "clear") || strings.EqualFold(val, "none") {
					delete(u.Metadata, "all")
					val = "Dihapus"
				} else {
					u.Metadata["all"] = val
				}
				_ = ext.DB.SaveUser(u)
				_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("✓ <b>Metadata Leech</b> diatur ke: <code>%s</code>", val), msg.MessageID, nil)
				return
			}
		case "split", "splitsize", "split_size":
			if len(parts) >= 3 {
				sizeStr := strings.ToUpper(parts[2])
				maxSplit := config.GetMaxSplitSize()
				var bytes int64
				if strings.HasSuffix(sizeStr, "GB") {
					val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStr, "GB"), 64)
					bytes = int64(val * 1024 * 1024 * 1024)
				} else if strings.HasSuffix(sizeStr, "MB") {
					val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStr, "MB"), 64)
					bytes = int64(val * 1024 * 1024)
				} else {
					bytes, _ = strconv.ParseInt(sizeStr, 10, 64)
				}
				if bytes > maxSplit || bytes <= 0 {
					bytes = maxSplit
				}
				u.SplitSize = bytes
				_ = ext.DB.SaveUser(u)
				_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("✓ <b>Ukuran Pemisahan Leech</b> diatur ke: <code>%s</code>", ext.GetReadableFileSize(u.SplitSize)), msg.MessageID, nil)
				return
			}
		case "leech":
			text, btn := m.buildLeechMenu(u, msg.SenderName)
			_, _ = m.sender.SendMessageWithButtons(ctx, peer, text, msg.MessageID, btn, 2)
			return
		}
	}

	// Default: Open Main Settings panel with one-by-one buttons
	text, btn := m.buildMainMenu(u, msg.SenderName)
	_, _ = m.sender.SendMessageWithButtons(ctx, peer, text, msg.MessageID, btn, 1)
}

func (m *SettingsModule) buildMainMenu(u *types.UserSettings, name string) (string, *telegram.ButtonMaker) {
	if name == "" {
		name = fmt.Sprintf("User_%d", u.UserID)
	}
	text := fmt.Sprintf("⌬ <b><u>%s</u> untuk <a href=\"tg://user?id=%d\">%s</a></b> (<code>%d</code>)\n\n"+
		"<i>Pilih kategori di bawah untuk mengonfigurasi preferensi Anda:</i>",
		telegram.ToSmallCaps("Pengaturan Pengguna"), u.UserID, name, u.UserID)

	btn := telegram.NewButtonMaker()
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Pengaturan Leech"), fmt.Sprintf("userset %d leech", u.UserID), "primary")
	btn.Row()
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Pengaturan Metadata"), fmt.Sprintf("userset %d metadata", u.UserID), "primary")
	btn.Row()
	btn.IButtonStyled("• "+telegram.ToSmallCaps("Atur Ulang Pengaturan"), fmt.Sprintf("userset %d reset_all", u.UserID), "danger")
	btn.Row()
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) buildLeechMenu(u *types.UserSettings, userName string) (string, *telegram.ButtonMaker) {
	if userName == "" {
		userName = fmt.Sprintf("User_%d", u.UserID)
	}
	nameTag := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", u.UserID, userName)

	// Daily leech usage and limit
	_, dailyBytes := ext.GetUserDailyStats(u.UserID)
	dlUsed := ext.GetReadableFileSize(dailyBytes)
	cfg := config.Get()
	dlLimit := "∞"
	if cfg.DailyLeechLimit > 0 {
		dlLimit = ext.GetReadableFileSize(int64(cfg.DailyLeechLimit) * 1024 * 1024 * 1024)
	}
	dlStr := fmt.Sprintf("%s / %s", dlUsed, dlLimit)

	ltype := "MEDIA"
	if u.AsDoc {
		ltype = "DOKUMEN"
	}

	thumbStatus := "Tidak Ada"
	if u.CustomThumb != "" {
		if _, err := os.Stat(u.CustomThumb); err == nil {
			thumbStatus = "Ada"
		}
	}

	splitSize := ext.GetReadableFileSize(config.GetMaxSplitSize()) + " (Bawaan)"
	if u.SplitSize > 0 {
		splitSize = ext.GetReadableFileSize(u.SplitSize)
	}

	equalSplits := "Nonaktif"
	if u.EqualSplits {
		equalSplits = "Aktif"
	}

	mediaGroup := "Nonaktif"
	if u.MediaGroup {
		mediaGroup = "Aktif"
	}

	caption := "Tidak Ada"
	if u.Caption != "" {
		caption = u.Caption
	}

	prefix := "Tidak Ada"
	if u.Prefix != "" {
		prefix = u.Prefix
	}

	suffix := "Tidak Ada"
	if u.Suffix != "" {
		suffix = u.Suffix
	}

	dumps := "Tidak Ada"
	if u.DumpChatID != 0 {
		dumps = fmt.Sprintf("%d", u.DumpChatID)
	}

	remnameAuto := "Tidak Ada"
	if u.RemnameAuto != "" {
		remnameAuto = u.RemnameAuto
	}

	remnameRegex := "Tidak Ada"
	if u.RemnameRegex != "" {
		remnameRegex = u.RemnameRegex
	}

	renameMethod := "Ganti Nama Otomatis"
	if u.RenameMethod == "regex" {
		renameMethod = "Regex"
	}

	autorename := "Nonaktif"
	if u.Autorename {
		autorename = "Aktif"
	}

	autoThumb := "Nonaktif"
	if u.AutoThumb {
		autoThumb = "Aktif"
	}

	metadata := "Tidak Ada"
	if u.Metadata != nil && u.Metadata["all"] != "" {
		metadata = u.Metadata["all"]
	}

	// Exact AZML text format with small caps header
	text := fmt.Sprintf("⌬ <b><u>%s :</u></b> %s\n\n"+
		"┎ <b>Leech Harian :</b> <code>%s</code> per hari\n"+
		"┠ <b>Jenis Leech :</b> <i>%s</i>\n"+
		"┠ <b>Thumbnail Kustom :</b> <i>%s</i>\n"+
		"┠ <b>Ukuran Pemisahan Leech :</b> <code>%s</code>\n"+
		"┠ <b>Pemisahan Sama Rata :</b> <i>%s</i>\n"+
		"┠ <b>Grup Media :</b> <i>%s</i>\n"+
		"┠ <b>Keterangan Leech :</b> <code>%s</code>\n"+
		"┠ <b>Awalan Leech :</b> <code>%s</code>\n"+
		"┠ <b>Akhiran Leech :</b> <code>%s</code>\n"+
		"┠ <b>Dump Leech :</b> <code>%s</code>\n"+
		"┠ <b>Templat AutoRename :</b> <code>%s</code>\n"+
		"┠ <b>Ganti Nama Regex :</b> <code>%s</code>\n"+
		"┠ <b>Metode Ganti Nama :</b> <i>%s</i>\n"+
		"┠ <b>AutoRename :</b> <i>%s</i>\n"+
		"┠ <b>Thumbnail Otomatis :</b> <i>%s</i>\n"+
		"┖ <b>Metadata Leech :</b> <code>%s</code>",
		telegram.ToSmallCaps("Pengaturan Leech"),
		nameTag, dlStr, ltype, thumbStatus, splitSize, equalSplits, mediaGroup,
		caption, prefix, suffix, dumps, remnameAuto, remnameRegex, renameMethod,
		autorename, autoThumb, metadata)

	btn := telegram.NewButtonMaker()

	// Row 1: Document/Media Toggle | Thumbnail
	if u.AsDoc {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Kirim Sebagai Media"), fmt.Sprintf("userset %d doc", u.UserID), "primary")
	} else {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Kirim Sebagai Dokumen"), fmt.Sprintf("userset %d doc", u.UserID), "primary")
	}
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Thumbnail"), fmt.Sprintf("userset %d thumb", u.UserID), "primary")
	btn.Row()

	// Row 2: Leech Splits | Leech Caption
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Pemisahan Leech"), fmt.Sprintf("userset %d split_size", u.UserID), "primary")
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Keterangan Leech"), fmt.Sprintf("userset %d lcaption", u.UserID), "primary")
	btn.Row()

	// Row 3: Leech Prefix | Leech Suffix
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Awalan Leech"), fmt.Sprintf("userset %d lprefix", u.UserID), "primary")
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Akhiran Leech"), fmt.Sprintf("userset %d lsuffix", u.UserID), "primary")
	btn.Row()

	// Row 4: AutoRename Template | Regex Remname
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Templat AutoRename"), fmt.Sprintf("userset %d lremname_auto", u.UserID), "primary")
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Ganti Nama Regex"), fmt.Sprintf("userset %d lremname_regex", u.UserID), "primary")
	btn.Row()

	// Row 5: Disable/Enable AutoRename | Enable/Disable Auto Thumbnail
	if u.Autorename {
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Nonaktifkan AutoRename"), fmt.Sprintf("userset %d autorename", u.UserID), "danger")
	} else {
		btn.IButtonStyled("✓ "+telegram.ToSmallCaps("Aktifkan AutoRename"), fmt.Sprintf("userset %d autorename", u.UserID), "success")
	}
	if u.AutoThumb {
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Nonaktifkan Thumbnail Otomatis"), fmt.Sprintf("userset %d auto_thumb", u.UserID), "danger")
	} else {
		btn.IButtonStyled("✓ "+telegram.ToSmallCaps("Aktifkan Thumbnail Otomatis"), fmt.Sprintf("userset %d auto_thumb", u.UserID), "success")
	}
	btn.Row()

	// Row 6: Leech Dump | Leech Metadata
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Dump Leech"), fmt.Sprintf("userset %d ldump", u.UserID), "primary")
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Metadata"), fmt.Sprintf("userset %d lmeta", u.UserID), "primary")
	btn.Row()

	// Row 7: Rename Method toggle
	if u.RenameMethod == "regex" {
		btn.IButtonStyled("• "+telegram.ToSmallCaps("Metode Ganti Nama: Regex"), fmt.Sprintf("userset %d rename_method", u.UserID), "primary")
	} else {
		btn.IButtonStyled("• "+telegram.ToSmallCaps("Metode Ganti Nama: Otomatis"), fmt.Sprintf("userset %d rename_method", u.UserID), "primary")
	}
	btn.Row()

	// Footer: Back | Close
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("userset %d main", u.UserID), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) buildThumbnailMenu(u *types.UserSettings, banner ...string) (string, *telegram.ButtonMaker) {
	status := "Tidak Ada"
	exists := false
	if u.CustomThumb != "" {
		if _, err := os.Stat(u.CustomThumb); err == nil {
			status = "Ada"
			exists = true
		}
	}

	bannerText := ""
	if len(banner) > 0 && banner[0] != "" {
		bannerText = banner[0] + "\n\n"
	}

	text := fmt.Sprintf("%s⌬ <b><u>%s</u></b>\n\n"+
		"┠ <b>Status:</b> <code>%s</code>\n"+
		"┖ <b>Berkas Tersimpan:</b> <code>%s</code>\n\n"+
		"<i>Thumbnail kustom akan disisipkan ke dalam berkas media hasil leech Anda.</i>\n\n"+
		"• Klik <b>Ubah Thumbnail</b> untuk mengaktifkan pemantau foto.\n"+
		"• Atau balas foto dengan <code>/us thumb</code>.",
		bannerText, telegram.ToSmallCaps("Pengaturan Thumbnail Kustom"), status, u.CustomThumb)

	btn := telegram.NewButtonMaker()
	if exists {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Lihat Thumbnail"), fmt.Sprintf("userset %d vthumb", u.UserID), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Hapus Thumbnail"), fmt.Sprintf("userset %d dthumb", u.UserID), "danger")
		btn.Row()
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Ubah Thumbnail"), fmt.Sprintf("userset %d cthumb", u.UserID), "primary")
		btn.Row()
	} else {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur Thumbnail"), fmt.Sprintf("userset %d cthumb", u.UserID), "success")
		btn.Row()
	}
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("userset %d leech", u.UserID), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) buildSplitSizeMenu(u *types.UserSettings) (string, *telegram.ButtonMaker) {
	splitSize := ext.GetReadableFileSize(config.GetMaxSplitSize()) + " (Bawaan)"
	if u.SplitSize > 0 {
		splitSize = ext.GetReadableFileSize(u.SplitSize)
	}

	eqStatus := "Nonaktif"
	if u.EqualSplits {
		eqStatus = "Aktif"
	}

	mgStatus := "Nonaktif"
	if u.MediaGroup {
		mgStatus = "Aktif"
	}

	text := fmt.Sprintf("⌬ <b><u>%s</u></b>\n\n"+
		"┠ <b>Ukuran Pemisahan Saat Ini:</b> <code>%s</code>\n"+
		"┠ <b>Pemisahan Sama Rata:</b> <i>%s</i>\n"+
		"┖ <b>Grup Media:</b> <i>%s</i>\n\n"+
		"<i>Berkas yang lebih besar dari batas ini akan dipecah otomatis sebelum diunggah.</i>",
		telegram.ToSmallCaps("Pengaturan Pemisahan Leech"), splitSize, eqStatus, mgStatus)

	btn := telegram.NewButtonMaker()

	// Toggles
	if u.EqualSplits {
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Nonaktifkan Pemisahan Sama Rata"), fmt.Sprintf("userset %d esplits", u.UserID), "danger")
	} else {
		btn.IButtonStyled("✓ "+telegram.ToSmallCaps("Aktifkan Pemisahan Sama Rata"), fmt.Sprintf("userset %d esplits", u.UserID), "success")
	}
	if u.MediaGroup {
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Nonaktifkan Grup Media"), fmt.Sprintf("userset %d mgroup", u.UserID), "danger")
	} else {
		btn.IButtonStyled("✓ "+telegram.ToSmallCaps("Aktifkan Grup Media"), fmt.Sprintf("userset %d mgroup", u.UserID), "success")
	}
	btn.Row()

	// Presets
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("2 GB"), fmt.Sprintf("userset %d split_2gb", u.UserID), "primary")
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("4 GB"), fmt.Sprintf("userset %d split_4gb", u.UserID), "primary")
	btn.IButtonStyled("• "+telegram.ToSmallCaps("Batas Bawaan"), fmt.Sprintf("userset %d split_default", u.UserID), "primary")
	btn.Row()

	// Custom
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Ukuran Kustom"), fmt.Sprintf("userset %d edit_split", u.UserID), "primary")
	btn.Row()
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("userset %d leech", u.UserID), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) buildCaptionMenu(u *types.UserSettings) (string, *telegram.ButtonMaker) {
	caption := "Tidak Ada (Bawaan)"
	if u.Caption != "" {
		caption = u.Caption
	}

	text := fmt.Sprintf("⌬ <b><u>%s</u></b>\n\n"+
		"┠ <b>Keterangan Saat Ini:</b>\n<code>%s</code>\n\n"+
		"➲ <b>Tag Format yang Didukung:</b>\n"+
		"• <code>{filename}</code> - Nama berkas asli atau yang sudah diganti\n"+
		"• <code>{size}</code> - Ukuran berkas (mis. 1.25 GB)\n"+
		"• <code>{duration}</code> - Durasi video (mis. 01:23:45)\n"+
		"• <code>{quality}</code> - Kualitas video (mis. 1080p, 720p)\n"+
		"• <code>{languages}</code> - Bahasa aliran audio\n"+
		"• <code>{subtitles}</code> - Bahasa subtitle",
		telegram.ToSmallCaps("Templat Keterangan Leech"), caption)

	btn := telegram.NewButtonMaker()
	if u.Caption != "" {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Ubah Keterangan"), fmt.Sprintf("userset %d edit_lcaption", u.UserID), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Atur Ulang ke Bawaan"), fmt.Sprintf("userset %d del_lcaption", u.UserID), "danger")
		btn.Row()
	} else {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur Keterangan Kustom"), fmt.Sprintf("userset %d edit_lcaption", u.UserID), "success")
		btn.Row()
	}
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("userset %d leech", u.UserID), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) buildPrefixMenu(u *types.UserSettings) (string, *telegram.ButtonMaker) {
	prefix := "Tidak Ada"
	if u.Prefix != "" {
		prefix = u.Prefix
	}

	text := fmt.Sprintf("⌬ <b><u>%s</u></b>\n\n"+
		"┠ <b>Awalan Saat Ini:</b> <code>%s</code>\n\n"+
		"➲ <b>Deskripsi:</b> <i>Teks yang ditambahkan di awal semua nama berkas hasil leech Anda.</i>\n\n"+
		"<b>Contoh:</b> <code>[AZML] </code>",
		telegram.ToSmallCaps("Pengaturan Awalan Leech"), prefix)

	btn := telegram.NewButtonMaker()
	if u.Prefix != "" {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Ubah Awalan"), fmt.Sprintf("userset %d edit_lprefix", u.UserID), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Hapus Awalan"), fmt.Sprintf("userset %d del_lprefix", u.UserID), "danger")
		btn.Row()
	} else {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur Awalan"), fmt.Sprintf("userset %d edit_lprefix", u.UserID), "success")
		btn.Row()
	}
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("userset %d leech", u.UserID), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) buildSuffixMenu(u *types.UserSettings) (string, *telegram.ButtonMaker) {
	suffix := "Tidak Ada"
	if u.Suffix != "" {
		suffix = u.Suffix
	}

	text := fmt.Sprintf("⌬ <b><u>%s</u></b>\n\n"+
		"┠ <b>Akhiran Saat Ini:</b> <code>%s</code>\n\n"+
		"➲ <b>Deskripsi:</b> <i>Teks yang ditambahkan sebelum ekstensi pada semua nama berkas hasil leech Anda.</i>\n\n"+
		"<b>Contoh:</b> <code> @MyChannel</code>",
		telegram.ToSmallCaps("Pengaturan Akhiran Leech"), suffix)

	btn := telegram.NewButtonMaker()
	if u.Suffix != "" {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Ubah Akhiran"), fmt.Sprintf("userset %d edit_lsuffix", u.UserID), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Hapus Akhiran"), fmt.Sprintf("userset %d del_lsuffix", u.UserID), "danger")
		btn.Row()
	} else {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur Akhiran"), fmt.Sprintf("userset %d edit_lsuffix", u.UserID), "success")
		btn.Row()
	}
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("userset %d leech", u.UserID), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) buildAutoRenameMenu(u *types.UserSettings) (string, *telegram.ButtonMaker) {
	tmpl := "Tidak Ada"
	if u.RemnameAuto != "" {
		tmpl = u.RemnameAuto
	}

	text := fmt.Sprintf("⌬ <b><u>%s</u></b>\n\n"+
		"┠ <b>Templat Saat Ini:</b>\n<code>%s</code>\n\n"+
		"➲ <b>Tag yang Didukung:</b>\n"+
		"• <code>{title}</code> - Judul media atau anime\n"+
		"• <code>{season}</code> - Nomor musim (mis. 01)\n"+
		"• <code>{episode}</code> - Nomor episode (mis. 05)\n"+
		"• <code>{quality}</code> - Resolusi yang terdeteksi (mis. 1080p)\n\n"+
		"<b>Contoh:</b>\n<code>{title} - S{season}E{episode} [{quality}]</code>",
		telegram.ToSmallCaps("Pengaturan Templat AutoRename"), tmpl)

	btn := telegram.NewButtonMaker()
	if u.RemnameAuto != "" {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Ubah Templat"), fmt.Sprintf("userset %d edit_lremname_auto", u.UserID), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Hapus Templat"), fmt.Sprintf("userset %d del_lremname_auto", u.UserID), "danger")
		btn.Row()
	} else {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur Templat"), fmt.Sprintf("userset %d edit_lremname_auto", u.UserID), "success")
		btn.Row()
	}
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("userset %d leech", u.UserID), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) buildRegexMenu(u *types.UserSettings) (string, *telegram.ButtonMaker) {
	regexVal := "Tidak Ada"
	if u.RemnameRegex != "" {
		regexVal = u.RemnameRegex
	}

	text := fmt.Sprintf("⌬ <b><u>%s</u></b>\n\n"+
		"┠ <b>Regex Saat Ini:</b>\n<code>%s</code>\n\n"+
		"➲ <b>Format:</b> <code>search_regex:replacement</code>\n"+
		"<b>Contoh:</b> <code>\\[TG\\]:</code> (menghapus [TG] dari nama berkas)",
		telegram.ToSmallCaps("Pengaturan Ganti Nama Regex"), regexVal)

	btn := telegram.NewButtonMaker()
	if u.RemnameRegex != "" {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Ubah Regex"), fmt.Sprintf("userset %d edit_lremname_regex", u.UserID), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Hapus Regex"), fmt.Sprintf("userset %d del_lremname_regex", u.UserID), "danger")
		btn.Row()
	} else {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur Regex"), fmt.Sprintf("userset %d edit_lremname_regex", u.UserID), "success")
		btn.Row()
	}
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("userset %d leech", u.UserID), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) buildDumpMenu(u *types.UserSettings) (string, *telegram.ButtonMaker) {
	dumpStr := "Tidak Ada"
	if u.DumpChatID != 0 {
		dumpStr = fmt.Sprintf("%d", u.DumpChatID)
	}

	text := fmt.Sprintf("⌬ <b><u>%s</u></b>\n\n"+
		"┠ <b>ID Kanal Dump:</b> <code>%s</code>\n\n"+
		"➲ <b>Deskripsi:</b> <i>Berkas hasil leech akan diteruskan ke kanal dump pribadi Anda.</i>\n\n"+
		"<b>Catatan:</b> Pastikan bot adalah <b>Administrator</b> di kanal Anda!",
		telegram.ToSmallCaps("Pengaturan Kanal Dump Leech"), dumpStr)

	btn := telegram.NewButtonMaker()
	if u.DumpChatID != 0 {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Ubah ID Dump"), fmt.Sprintf("userset %d edit_ldump", u.UserID), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Hapus ID Dump"), fmt.Sprintf("userset %d del_ldump", u.UserID), "danger")
		btn.Row()
	} else {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur ID Dump"), fmt.Sprintf("userset %d edit_ldump", u.UserID), "success")
		btn.Row()
	}
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("userset %d leech", u.UserID), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) buildMetadataMenu(u *types.UserSettings) (string, *telegram.ButtonMaker) {
	metaAll := "Tidak Ada"
	if u.Metadata != nil && u.Metadata["all"] != "" {
		metaAll = u.Metadata["all"]
	}

	text := fmt.Sprintf("⌬ <b><u>%s</u></b>\n\n"+
		"┠ <b>Teks Metadata:</b> <code>%s</code>\n\n"+
		"➲ <b>Deskripsi:</b> <i>Judul dan penulis metadata yang disisipkan langsung ke aliran video melalui FFmpeg.</i>\n\n"+
		"<b>Contoh:</b> <code>Encoded by TBX TEAM</code>",
		telegram.ToSmallCaps("Pengaturan Metadata Video Leech"), metaAll)

	btn := telegram.NewButtonMaker()
	if metaAll != "Tidak Ada" {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Ubah Metadata"), fmt.Sprintf("userset %d edit_lmeta", u.UserID), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Hapus Metadata"), fmt.Sprintf("userset %d del_lmeta", u.UserID), "danger")
		btn.Row()
	} else {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur Metadata"), fmt.Sprintf("userset %d edit_lmeta", u.UserID), "success")
		btn.Row()
	}
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("userset %d leech", u.UserID), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
	btn.Row()

	return text, btn
}

func (m *SettingsModule) HandleCallback(ctx context.Context, peer tg.InputPeerClass, queryID int64, msgID int, data string, fromUserID int64) {
	parts := strings.Fields(data)
	if len(parts) >= 2 {
		targetUID, err := strconv.ParseInt(parts[1], 10, 64)
		if err == nil && targetUID != fromUserID {
			_ = m.sender.AnswerCallback(ctx, queryID, "⚠️ Ini bukan panel pengaturan Anda! Gunakan /us untuk membuka panel Anda sendiri.", true)
			return
		}
	}
	if len(parts) < 3 {
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)
		return
	}

	targetUID, _ := strconv.ParseInt(parts[1], 10, 64)
	if targetUID != fromUserID {
		_ = m.sender.AnswerCallback(ctx, queryID, "⚠️ Ini bukan panel pengaturan Anda! Gunakan /us untuk membuka panel Anda sendiri.", true)
		return
	}

	action := parts[2]
	u := m.userMap.GetOrCreate(targetUID)

	switch action {
	// 1. Navigation
	case "main", "back":
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)
		text, btn := m.buildMainMenu(u, "")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 1)
	case "leech":
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)
		text, btn := m.buildLeechMenu(u, "")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
	case "reset_all":
		u.Prefix = ""
		u.Suffix = ""
		u.Caption = ""
		u.RemnameAuto = ""
		u.RemnameRegex = ""
		u.SplitSize = 0
		u.Autorename = false
		u.AutoThumb = false
		u.EqualSplits = false
		u.MediaGroup = false
		u.AsDoc = config.Get().AsDocument
		if u.CustomThumb != "" {
			_ = os.Remove(u.CustomThumb)
			u.CustomThumb = ""
		}
		go func(usr *types.UserSettings) { _ = ext.DB.SaveUser(usr) }(u)
		_ = m.sender.AnswerCallback(ctx, queryID, "Semua pengaturan pengguna diatur ulang ke bawaan!", true)
		text, btn := m.buildMainMenu(u, "")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 1)
	case "close":
		ext.ClearPendingSettingSession(targetUID)
		ext.ClearPendingThumbSession(targetUID)
		_ = m.sender.AnswerCallback(ctx, queryID, "Pengaturan ditutup", false)
		_ = m.sender.DeleteMessage(ctx, peer, []int{msgID})

	// 2. Direct Toggles (Edit message in-place without glitch)
	case "doc":
		u.AsDoc = !u.AsDoc
		go func(usr *types.UserSettings) { _ = ext.DB.SaveUser(usr) }(u)
		st := "MEDIA"
		if u.AsDoc {
			st = "DOKUMEN"
		}
		_ = m.sender.AnswerCallback(ctx, queryID, fmt.Sprintf("Jenis unggah diatur ke %s", st), false)
		text, btn := m.buildLeechMenu(u, "")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)

	case "autorename":
		u.Autorename = !u.Autorename
		go func(usr *types.UserSettings) { _ = ext.DB.SaveUser(usr) }(u)
		st := "Nonaktif"
		if u.Autorename {
			st = "Aktif"
		}
		_ = m.sender.AnswerCallback(ctx, queryID, fmt.Sprintf("AutoRename sekarang %s", st), false)
		text, btn := m.buildLeechMenu(u, "")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)

	case "auto_thumb":
		u.AutoThumb = !u.AutoThumb
		go func(usr *types.UserSettings) { _ = ext.DB.SaveUser(usr) }(u)
		st := "Nonaktif"
		if u.AutoThumb {
			st = "Aktif"
		}
		_ = m.sender.AnswerCallback(ctx, queryID, fmt.Sprintf("Thumbnail otomatis sekarang %s", st), false)
		text, btn := m.buildLeechMenu(u, "")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)

	case "rename_method":
		if u.RenameMethod == "auto" {
			u.RenameMethod = "regex"
		} else {
			u.RenameMethod = "auto"
		}
		go func(usr *types.UserSettings) { _ = ext.DB.SaveUser(usr) }(u)
		_ = m.sender.AnswerCallback(ctx, queryID, fmt.Sprintf("Metode: %s", strings.Title(u.RenameMethod)), false)
		text, btn := m.buildLeechMenu(u, "")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)

	case "esplits":
		u.EqualSplits = !u.EqualSplits
		go func(usr *types.UserSettings) { _ = ext.DB.SaveUser(usr) }(u)
		st := "Nonaktif"
		if u.EqualSplits {
			st = "Aktif"
		}
		_ = m.sender.AnswerCallback(ctx, queryID, fmt.Sprintf("Pemisahan sama rata sekarang %s", st), false)
		text, btn := m.buildSplitSizeMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)

	case "mgroup":
		u.MediaGroup = !u.MediaGroup
		go func(usr *types.UserSettings) { _ = ext.DB.SaveUser(usr) }(u)
		st := "Nonaktif"
		if u.MediaGroup {
			st = "Aktif"
		}
		_ = m.sender.AnswerCallback(ctx, queryID, fmt.Sprintf("Grup media sekarang %s", st), false)
		text, btn := m.buildSplitSizeMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)

	// 3. Thumbnail Submenu
	case "thumb":
		ext.ClearPendingThumbSession(targetUID)
		text, btn := m.buildThumbnailMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "cthumb":
		chatID := telegram.PeerToChatID(peer)
		ext.SetPendingThumbSession(targetUID, chatID, msgID, 120*time.Second)
		prompt := "⌬ <b><u>Kirim Thumbnail Kustom</u></b>\n\n" +
			"Silakan kirim <b>foto</b> atau dokumen gambar dalam waktu <b>2 menit</b>.\n\n" +
			"<i>Setelah diterima, berkas akan disimpan otomatis sebagai thumbnail kustom Anda dan menu ini akan diperbarui.</i>"
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Batalkan"), fmt.Sprintf("userset %d thumb", u.UserID), "danger")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, prompt, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim foto Anda...", false)

	case "vthumb":
		if u.CustomThumb == "" {
			_ = m.sender.AnswerCallback(ctx, queryID, "Thumbnail belum diatur!", true)
			return
		}
		if _, err := os.Stat(u.CustomThumb); err != nil {
			_ = m.sender.AnswerCallback(ctx, queryID, "Berkas thumbnail tidak ditemukan di disk!", true)
			return
		}
		_, err := m.sender.SendLocalPhoto(ctx, peer, u.CustomThumb, "⌬ <b>Thumbnail Kustom Anda</b>", 0)
		if err != nil {
			_ = m.sender.AnswerCallback(ctx, queryID, fmt.Sprintf("Kesalahan saat mengirim thumbnail: %v", err), true)
		} else {
			_ = m.sender.AnswerCallback(ctx, queryID, "Thumbnail terkirim!", false)
		}

	case "dthumb":
		ext.ClearPendingThumbSession(targetUID)
		if u.CustomThumb != "" {
			_ = os.Remove(u.CustomThumb)
		}
		dest := fmt.Sprintf("Thumbnails/%d.jpg", targetUID)
		_ = os.Remove(dest)
		u.CustomThumb = ""
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildThumbnailMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "✓ Thumbnail kustom dihapus!", false)

	// 4. Split Size Submenu
	case "split_size":
		text, btn := m.buildSplitSizeMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "split_2gb":
		u.SplitSize = 2000 * 1024 * 1024
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildSplitSizeMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Ukuran pemisahan diatur ke 2GB", false)

	case "split_4gb":
		u.SplitSize = 4000 * 1024 * 1024
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildSplitSizeMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Ukuran pemisahan diatur ke 4GB", false)

	case "split_default":
		u.SplitSize = 0
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildSplitSizeMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Diatur ulang ke batas bawaan", false)

	case "edit_split":
		ext.SetPendingSettingSession(targetUID, "split_size", 0, msgID, 60*time.Second)
		prompt := "⌬ <b><u>Atur Ukuran Pemisahan Kustom</u></b>\n\n" +
			"Kirim ukuran kustom (mis. <code>1.5GB</code>, <code>500MB</code>, <code>2000000000</code>) dalam waktu <b>60 detik</b>:"
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Batalkan"), fmt.Sprintf("userset %d split_size", u.UserID), "danger")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, prompt, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim ukuran pemisahan baru...", false)

	// 5. Caption Submenu
	case "lcaption":
		text, btn := m.buildCaptionMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "edit_lcaption":
		ext.SetPendingSettingSession(targetUID, "lcaption", 0, msgID, 60*time.Second)
		prompt := "⌬ <b><u>Atur Keterangan Leech Kustom</u></b>\n\n" +
			"Kirim templat keterangan baru dalam waktu <b>60 detik</b>.\n\n" +
			"<b>Tag:</b> <code>{filename}</code>, <code>{size}</code>, <code>{duration}</code>, <code>{quality}</code>, <code>{languages}</code>"
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Batalkan"), fmt.Sprintf("userset %d lcaption", u.UserID), "danger")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, prompt, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim templat keterangan baru...", false)

	case "del_lcaption":
		u.Caption = ""
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildCaptionMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Keterangan diatur ulang ke bawaan", false)

	// 6. Prefix Submenu
	case "lprefix":
		text, btn := m.buildPrefixMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "edit_lprefix":
		ext.SetPendingSettingSession(targetUID, "lprefix", 0, msgID, 60*time.Second)
		prompt := "⌬ <b><u>Atur Awalan Leech</u></b>\n\n" +
			"Kirim teks awalan yang akan ditambahkan ke nama berkas dalam waktu <b>60 detik</b> (mis. <code>[AZML] </code>):"
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Batalkan"), fmt.Sprintf("userset %d lprefix", u.UserID), "danger")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, prompt, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim awalan baru...", false)

	case "del_lprefix":
		u.Prefix = ""
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildPrefixMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Awalan dihapus", false)

	// 7. Suffix Submenu
	case "lsuffix":
		text, btn := m.buildSuffixMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "edit_lsuffix":
		ext.SetPendingSettingSession(targetUID, "lsuffix", 0, msgID, 60*time.Second)
		prompt := "⌬ <b><u>Atur Akhiran Leech</u></b>\n\n" +
			"Kirim teks akhiran yang akan ditambahkan sebelum ekstensi berkas dalam waktu <b>60 detik</b> (mis. <code> @MyChannel</code>):"
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Batalkan"), fmt.Sprintf("userset %d lsuffix", u.UserID), "danger")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, prompt, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim akhiran baru...", false)

	case "del_lsuffix":
		u.Suffix = ""
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildSuffixMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Akhiran dihapus", false)

	// 8. AutoRename Template Submenu
	case "lremname_auto":
		text, btn := m.buildAutoRenameMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "edit_lremname_auto":
		ext.SetPendingSettingSession(targetUID, "lremname_auto", 0, msgID, 60*time.Second)
		prompt := "⌬ <b><u>Atur Templat AutoRename</u></b>\n\n" +
			"Kirim templat penggantian nama dalam waktu <b>60 detik</b>.\n\n" +
			"<b>Tag:</b> <code>{title}</code>, <code>{season}</code>, <code>{episode}</code>, <code>{quality}</code>\n" +
			"<b>Contoh:</b> <code>{title} - S{season}E{episode} [{quality}]</code>"
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Batalkan"), fmt.Sprintf("userset %d lremname_auto", u.UserID), "danger")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, prompt, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim templat AutoRename baru...", false)

	case "del_lremname_auto":
		u.RemnameAuto = ""
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildAutoRenameMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Templat AutoRename dihapus", false)

	// 9. Regex Remname Submenu
	case "lremname_regex":
		text, btn := m.buildRegexMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "edit_lremname_regex":
		ext.SetPendingSettingSession(targetUID, "lremname_regex", 0, msgID, 60*time.Second)
		prompt := "⌬ <b><u>Atur Ganti Nama Regex</u></b>\n\n" +
			"Kirim pola regex dan penggantinya yang dipisahkan oleh titik dua dalam waktu <b>60 detik</b>.\n\n" +
			"<b>Format:</b> <code>search_pattern:replacement</code>\n" +
			"<b>Contoh:</b> <code>\\[TG\\]:</code>"
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Batalkan"), fmt.Sprintf("userset %d lremname_regex", u.UserID), "danger")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, prompt, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim aturan regex baru...", false)

	case "del_lremname_regex":
		u.RemnameRegex = ""
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildRegexMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Aturan regex dihapus", false)

	// 10. Dump Channel Submenu
	case "ldump":
		text, btn := m.buildDumpMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "edit_ldump":
		ext.SetPendingSettingSession(targetUID, "ldump", 0, msgID, 60*time.Second)
		prompt := "⌬ <b><u>Atur ID Kanal Dump Leech</u></b>\n\n" +
			"Kirim ID kanal Anda dalam waktu <b>60 detik</b> (mis. <code>-1001234567890</code>).\n\n" +
			"<i>Pastikan bot memiliki hak Admin dengan izin Posting Pesan!</i>"
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Batalkan"), fmt.Sprintf("userset %d ldump", u.UserID), "danger")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, prompt, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim ID Kanal Dump...", false)

	case "del_ldump":
		u.DumpChatID = 0
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildDumpMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kanal dump dihapus", false)

	// 11. Metadata Submenu
	case "lmeta", "metadata":
		text, btn := m.buildMetadataMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "edit_lmeta":
		ext.SetPendingSettingSession(targetUID, "lmeta", 0, msgID, 60*time.Second)
		prompt := "⌬ <b><u>Atur Metadata Video Leech</u></b>\n\n" +
			"Kirim teks metadata dalam waktu <b>60 detik</b> (mis. <code>Encoded by TBX TEAM</code>):"
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Batalkan"), fmt.Sprintf("userset %d lmeta", u.UserID), "danger")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("userset %d close", u.UserID), "danger")
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, prompt, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim teks metadata...", false)

	case "del_lmeta":
		if u.Metadata != nil {
			delete(u.Metadata, "all")
		}
		_ = ext.DB.SaveUser(u)
		text, btn := m.buildMetadataMenu(u)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Metadata dihapus", false)

	default:
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)
	}
}

// HandleSettingUpdated refreshes and edits the prompt message back to the corresponding submenu
func (m *SettingsModule) HandleSettingUpdated(ctx context.Context, peer tg.InputPeerClass, promptMsgID int, u *types.UserSettings, key string) {
	if promptMsgID <= 0 || u == nil {
		return
	}
	var text string
	var btn *telegram.ButtonMaker
	switch key {
	case "lprefix":
		text, btn = m.buildPrefixMenu(u)
	case "lsuffix":
		text, btn = m.buildSuffixMenu(u)
	case "lcaption":
		text, btn = m.buildCaptionMenu(u)
	case "lremname_auto":
		text, btn = m.buildAutoRenameMenu(u)
	case "lremname_regex":
		text, btn = m.buildRegexMenu(u)
	case "ldump":
		text, btn = m.buildDumpMenu(u)
	case "lmeta", "metadata":
		text, btn = m.buildMetadataMenu(u)
	case "split_size", "edit_split":
		text, btn = m.buildSplitSizeMenu(u)
	case "thumb", "cthumb":
		text, btn = m.buildThumbnailMenu(u, "✓ <b>Thumbnail kustom berhasil disimpan!</b>")
	default:
		text, btn = m.buildLeechMenu(u, "")
	}
	_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
}

func (m *SettingsModule) HandleAutoRename(ctx context.Context, peer tg.InputPeerClass, msg *types.MessageContext) {
	u := m.userMap.GetOrCreate(msg.UserID)
	parts := strings.SplitN(strings.TrimSpace(msg.Text), " ", 2)

	// --- No argument: show current template or help ---
	if len(parts) == 1 || strings.TrimSpace(parts[1]) == "" {
		current := u.RemnameAuto
		if current == "" {
			current = config.Get().LeechFilenameRemnameAuto
		}

		if current != "" {
			text := fmt.Sprintf("⌬ <b><u>%s :</u></b>\n\n"+
				"<code>%s</code>\n\n"+
				"<b>Penggunaan:</b> <code>/ar [template]</code>\n"+
				"<b>Hapus:</b> <code>/ar off</code>",
				telegram.ToSmallCaps("Templat AutoRename Saat Ini"),
				telegram.EscapeHTML(current))
			_, _ = m.sender.SendMessage(ctx, peer, text, msg.MessageID, nil)
		} else {
			text := fmt.Sprintf("⌬ <b><u>%s :</u></b>\n\n"+
				"<b>Templat AutoRename belum diatur.</b>\n\n"+
				"<b>Penggunaan:</b> <code>/ar [animeshrine.xyz] S0{season}E{episode} Dr. Stone {quality}.mkv</code>\n\n"+
				"<b>Tag yang tersedia:</b>\n"+
				"  • <code>{title}</code> — judul hasil parsing\n"+
				"  • <code>{season}</code> — nomor musim\n"+
				"  • <code>{episode}</code> — nomor episode\n"+
				"  • <code>{quality}</code> — kualitas video (mis. 1080p)\n"+
				"  • <code>{codec}</code> — codec video\n"+
				"  • <code>{audio}</code> — codec audio\n"+
				"  • <code>{chapter}</code> — nomor bab\n"+
				"  • <code>{episode:+N}</code> — episode dengan offset\n\n"+
				"<b>Hapus:</b> <code>/ar off</code>",
				telegram.ToSmallCaps("Panduan AutoRename"))
			_, _ = m.sender.SendMessage(ctx, peer, text, msg.MessageID, nil)
		}
		return
	}

	template := strings.TrimSpace(parts[1])

	// --- Clear template ---
	switch strings.ToLower(template) {
	case "off", "clear", "reset", "del", "none":
		u.RemnameAuto = ""
		_ = ext.DB.SaveUser(u)
		_, _ = m.sender.SendMessage(ctx, peer, "✓ <b>Templat AutoRename dihapus!</b>", msg.MessageID, nil)
		return

	default:
		u.RemnameAuto = template
		u.Autorename = true
		u.RenameMethod = "auto"
		_ = ext.DB.SaveUser(u)
		text := fmt.Sprintf("✓ <b>Templat AutoRename disimpan!</b>\n\n<code>%s</code>", telegram.EscapeHTML(template))
		_, _ = m.sender.SendMessage(ctx, peer, text, msg.MessageID, nil)
		return
	}
}
