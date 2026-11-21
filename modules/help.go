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

	"github.com/gotd/td/tg"
)

type HelpModule struct {
	sender    *telegram.MessageSender
	userMap   *types.SafeUserMap
	startTime time.Time
}

func NewHelpModule(sender *telegram.MessageSender, userMap *types.SafeUserMap, startTime time.Time) *HelpModule {
	return &HelpModule{
		sender:    sender,
		userMap:   userMap,
		startTime: startTime,
	}
}

func (m *HelpModule) getStartText(name string) string {
	if name == "" {
		name = "Tuan"
	}
	return fmt.Sprintf("<blockquote>%s, <b>%s</b></blockquote>\n\n"+
		"<blockquote>%s</blockquote>\n\n"+
		"<blockquote>❝ <b>%s</b> ❞\n\n"+
		"― %s</blockquote>",
		telegram.ToSmallCaps("Halo"),
		name,
		telegram.ToSmallCaps("Saya adalah Leech AZML-Go dengan Bot Auto Rename, dibuat dengan penuh cinta."),
		telegram.ToSmallCaps("Menang adalah segalanya. Selama saya menang pada akhirnya, itu saja yang penting."),
		telegram.ToSmallCaps("Cinta dari Go"))
}

func (m *HelpModule) getStartButtons(ownerID int64) *telegram.ButtonMaker {
	cfg := config.Get()
	btn := telegram.NewButtonMaker()
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Panduan Bantuan"), "start_help", "primary")

	ownerUsername := strings.TrimPrefix(strings.TrimSpace(cfg.OwnerUsername), "@")
	if ownerUsername == "" {
		ownerUsername = "exe_aquib"
	}
	ownerURL := fmt.Sprintf("https://t.me/%s", ownerUsername)

	btn.UButtonStyled("◈ "+telegram.ToSmallCaps("Pemilik"), ownerURL, "primary")
	btn.Row()
	return btn
}

func (m *HelpModule) getHelpGuideText() string {
	cfg := config.Get()
	sfx := cfg.CmdSuffix
	return fmt.Sprintf("⌬ <b><u>%s :</u></b>\n\n"+
		"┎ <b>Leech & Mirror:</b>\n"+
		"┠ <code>/leech%s [link]</code> — Langsung, Magnet, Torrent, Mega, media TG\n"+
		"┠ <code>/zipleech%s [link]</code> — Zip konten sebelum mengunggah\n"+
		"┠ <code>/unzipleech%s [link]</code> — Ekstrak arsip zip/rar/7z\n"+
		"┖ <code>/ytdl%s [link]</code> — Unduh video/audio melalui yt-dlp\n\n"+
		"┎ <b>Pengaturan Pengguna:</b>\n"+
		"┠ <code>/us%s</code> — Pengaturan pengguna (Awalan, Akhiran, Keterangan, AutoRename)\n"+
		"┠ <code>/ar%s [template]</code> — Atur/hapus templat AutoRename cepat\n"+
		"┠ <code>/setthumb%s</code> (balas) — Atur thumbnail leech kustom\n"+
		"┖ <code>/status%s</code> — Kecepatan dan progres unduh & unggah secara langsung\n\n"+
		"┎ <b>Admin & Kontrol:</b>\n"+
		"┠ <code>/bs%s</code> — Pengaturan Bot (variabel konfigurasi, Aria2c, berkas pribadi)\n"+
		"┠ <code>/cancel%s [GID]</code> — Batalkan tugas tertentu\n"+
		"┖ <code>/cancelall%s</code> — Batalkan semua tugas aktif Anda\n\n"+
		"<i>Klik Kembali di bawah untuk kembali ke menu awal.</i>",
		telegram.ToSmallCaps("Panduan Perintah AZML-Go"),
		sfx, sfx, sfx, sfx, sfx, sfx, sfx, sfx, sfx, sfx, sfx)
}

func (m *HelpModule) HandleStart(ctx context.Context, peer tg.InputPeerClass, msg *types.MessageContext) {
	u := m.userMap.GetOrCreate(msg.UserID)
	u.StartedInPM = true
	if msg.SenderName != "" {
		u.Name = msg.SenderName
	}
	_ = ext.DB.SaveUser(u)

	tokens := strings.Fields(msg.Text)
	if len(tokens) > 1 {
		tokenArg := tokens[1]
		if ok, reply := ext.VerifyToken(tokenArg, msg.UserID, m.userMap); ok {
			_, _ = m.sender.SendMessage(ctx, peer, reply, msg.MessageID, nil)
			return
		}
	}

	cfg := config.Get()
	startText := m.getStartText(msg.SenderName)
	startBtns := m.getStartButtons(cfg.OwnerID)

	cover := cfg.CoverImage
	if cover != "" && (strings.HasPrefix(cover, "http://") || strings.HasPrefix(cover, "https://")) {
		_, err := m.sender.SendPhotoFromURLWithButtons(ctx, peer, cover, startText, msg.MessageID, startBtns, 2)
		if err == nil {
			return
		}
	}

	_, _ = m.sender.SendMessageWithButtons(ctx, peer, startText, msg.MessageID, startBtns, 2)
}

func (m *HelpModule) HandleHelp(ctx context.Context, peer tg.InputPeerClass, msg *types.MessageContext) {
	helpText := m.getHelpGuideText()
	btn := telegram.NewButtonMaker()
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), "start_close", "danger")
	btn.Row()
	_, _ = m.sender.SendMessageWithButtons(ctx, peer, helpText, msg.MessageID, btn, 1)
}

func (m *HelpModule) HandlePing(ctx context.Context, peer tg.InputPeerClass, msg *types.MessageContext) {
	start := time.Now()
	resID, err := m.sender.SendMessage(ctx, peer, "<i>Mengukur ping...</i>", msg.MessageID, nil)
	if err == nil && resID > 0 {
		diff := time.Since(start).Milliseconds()
		_ = m.sender.EditMessage(ctx, peer, resID, fmt.Sprintf("⌬ <b>Pong!</b> <code>%d ms</code>", diff), nil)
	}
}

func (m *HelpModule) getStatsView(key string, targetUID int64) (string, *telegram.ButtonMaker) {
	btn := telegram.NewButtonMaker()
	var msg string

	if key == "home" {
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Statistik Bot"), fmt.Sprintf("stats %d stbot", targetUID), "primary")
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Statistik Sistem"), fmt.Sprintf("stats %d stsys", targetUID), "primary")
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Statistik Repo"), fmt.Sprintf("stats %d strepo", targetUID), "primary")
		btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Batasan Bot"), fmt.Sprintf("stats %d botlimits", targetUID), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("stats %d close", targetUID), "danger")
		msg = "⌬ <b><i>Statistik Bot & Sistem!</i></b>"
	} else {
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("stats %d home", targetUID), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), fmt.Sprintf("stats %d close", targetUID), "danger")

		switch key {
		case "stbot":
			msg = ext.RenderBotStats(m.startTime)
		case "stsys":
			msg = ext.RenderSysStats()
		case "strepo":
			msg = ext.RenderRepoStats()
		case "botlimits":
			msg = ext.RenderBotLimits()
		default:
			msg = "⌬ <b><i>Statistik Bot & Sistem!</i></b>"
		}
	}

	return msg, btn
}

func (m *HelpModule) HandleStats(ctx context.Context, peer tg.InputPeerClass, msg *types.MessageContext) {
	text, btn := m.getStatsView("home", msg.UserID)
	_, _ = m.sender.SendMessageWithButtons(ctx, peer, text, msg.MessageID, btn, 2)
}

func (m *HelpModule) HandleStatsCallback(ctx context.Context, peer tg.InputPeerClass, queryID int64, msgID int, data string, fromUID int64) {
	if data == "start_help" {
		helpText := m.getHelpGuideText()
		helpBtn := telegram.NewButtonMaker()
		helpBtn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), "start_back", "primary")
		helpBtn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), "start_close", "danger")
		helpBtn.Row()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, helpText, helpBtn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)
		return
	}
	if data == "start_back" {
		cfg := config.Get()
		name := "Tuan"
		if u := m.userMap.Get(fromUID); u != nil && u.Name != "" {
			name = u.Name
		}
		startText := m.getStartText(name)
		startBtns := m.getStartButtons(cfg.OwnerID)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, startText, startBtns, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)
		return
	}
	if data == "start_close" {
		_ = m.sender.DeleteMessage(ctx, peer, []int{msgID})
		_ = m.sender.AnswerCallback(ctx, queryID, "Ditutup", false)
		return
	}

	parts := strings.Fields(data)
	if len(parts) < 3 {
		return
	}

	var targetUID int64
	var key string

	if parts[0] == "wzmlx" {
		targetUID, _ = strconv.ParseInt(parts[1], 10, 64)
		if parts[2] == "close" {
			key = "close"
		} else if len(parts) >= 4 && parts[2] == "stats" {
			key = parts[3]
		}
	} else if parts[0] == "stats" {
		targetUID, _ = strconv.ParseInt(parts[1], 10, 64)
		key = parts[2]
	}

	if targetUID != 0 && targetUID != fromUID && !telegram.IsOwner(fromUID) {
		_ = m.sender.AnswerCallback(ctx, queryID, "Menu ini bukan untuk Anda!", true)
		return
	}

	if key == "close" {
		_ = m.sender.DeleteMessage(ctx, peer, []int{msgID})
		_ = m.sender.AnswerCallback(ctx, queryID, "Ditutup!", false)
		return
	}

	text, btn := m.getStatsView(key, targetUID)
	_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
	_ = m.sender.AnswerCallback(ctx, queryID, "", false)
}

func (m *HelpModule) HandleLog(ctx context.Context, peer tg.InputPeerClass, msg *types.MessageContext) {
	if !telegram.IsOwner(msg.UserID) && !telegram.IsSudo(msg.UserID, m.userMap) {
		_, _ = m.sender.SendMessage(ctx, peer, "Hanya Pemilik/Sudo yang dapat melihat log!", msg.MessageID, nil)
		return
	}
	_, _ = m.sender.SendMessage(ctx, peer, "⌬ <b>Log:</b> Semua layanan beroperasi dengan normal.", msg.MessageID, nil)
}

func (m *HelpModule) HandleRestart(ctx context.Context, peer tg.InputPeerClass, msg *types.MessageContext) {
	if !telegram.IsOwner(msg.UserID) {
		_, _ = m.sender.SendMessage(ctx, peer, "Hanya Pemilik yang dapat memulai ulang bot!", msg.MessageID, nil)
		return
	}

	statusMsgID, _ := m.sender.SendMessage(ctx, peer, "• <i>Menginisialisasi mesin Pembaruan & Mulai Ulang AZML-Go...</i>", msg.MessageID, nil)

	onStatus := func(text string) {
		if statusMsgID > 0 {
			_ = m.sender.EditMessage(ctx, peer, statusMsgID, text, nil)
		} else {
			_, _ = m.sender.SendMessage(ctx, peer, text, msg.MessageID, nil)
		}
	}

	recordRestartMsg := func() error {
		if statusMsgID > 0 {
			return os.WriteFile(".restartmsg", []byte(fmt.Sprintf("%d %d", msg.ChatID, statusMsgID)), 0644)
		}
		return nil
	}

	go func() {
		_ = ext.UpdateAndRestart(context.Background(), onStatus, recordRestartMsg)
	}()
}
