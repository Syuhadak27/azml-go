package modules

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"azml-go/config"
	"azml-go/helper/ext"
	"azml-go/helper/telegram"
	"azml-go/helper/uploader"
	"azml-go/types"

	"github.com/gotd/td/tg"
)

type BotSettingsModule struct {
	sender              *telegram.MessageSender
	userMap             *types.SafeUserMap
	userSessionReloader func(ctx context.Context, sessionStr string) (string, int64, bool, error)
	uploader            *uploader.TelegramUploader
	leechMod            *LeechModule
}

func NewBotSettingsModule(sender *telegram.MessageSender, userMap *types.SafeUserMap) *BotSettingsModule {
	return &BotSettingsModule{
		sender:  sender,
		userMap: userMap,
	}
}

func (m *BotSettingsModule) SetUserSessionReloader(fn func(ctx context.Context, sessionStr string) (string, int64, bool, error)) {
	m.userSessionReloader = fn
}

func (m *BotSettingsModule) SetUploader(u *uploader.TelegramUploader) {
	m.uploader = u
}

func (m *BotSettingsModule) SetLeechModule(l *LeechModule) {
	m.leechMod = l
}

var configVarKeys = []string{
	"OWNER_USERNAME",
	"USER_SESSION_STRING",
	"DUMP_CHAT_ID",
	"LEECH_LOG_ID",
	"STATUS_UPDATE_INTERVAL",
	"STATUS_LIMIT",
	"LEECH_SPLIT_SIZE",
	"AS_DOCUMENT",
	"AUTO_THUMBNAIL",
	"EQUAL_SPLITS",
	"MEDIA_GROUP",
	"LEECH_FILENAME_PREFIX",
	"LEECH_FILENAME_SUFFIX",
	"LEECH_FILENAME_CAPTION",
	"TORRENT_TIMEOUT",
	"BASE_URL",
	"BOT_MAX_TASKS",
	"USER_MAX_TASKS",
	"LEECH_LIMIT",
	"TORRENT_LIMIT",
	"DIRECT_LIMIT",
	"YTDLP_LIMIT",
	"MEGA_LIMIT",
	"DAILY_LEECH_LIMIT",
	"DAILY_TASK_LIMIT",
	"AUTO_DELETE_MESSAGE_DURATION",
	"SHORTENER",
	"SHORTENER_API",
	"SHORTENER_ENABLED",
	"TOKEN_TIMEOUT",
	"UPSTREAM_REPO",
	"UPSTREAM_BRANCH",
}

var defaultVarValues = map[string]string{
	"OWNER_USERNAME":               "exe_aquib",
	"USER_SESSION_STRING":          "",
	"DUMP_CHAT_ID":                 "0",
	"LEECH_LOG_ID":                 "0",
	"STATUS_UPDATE_INTERVAL":       "10",
	"STATUS_LIMIT":                 "4",
	"LEECH_SPLIT_SIZE":             "0",
	"AS_DOCUMENT":                  "false",
	"AUTO_THUMBNAIL":               "false",
	"EQUAL_SPLITS":                 "false",
	"MEDIA_GROUP":                  "false",
	"LEECH_FILENAME_PREFIX":        "",
	"LEECH_FILENAME_SUFFIX":        "",
	"LEECH_FILENAME_CAPTION":       "",
	"TORRENT_TIMEOUT":              "0",
	"BASE_URL":                     "",
	"BOT_MAX_TASKS":                "0",
	"USER_MAX_TASKS":               "0",
	"LEECH_LIMIT":                  "0",
	"TORRENT_LIMIT":                "0",
	"DIRECT_LIMIT":                 "0",
	"YTDLP_LIMIT":                  "0",
	"MEGA_LIMIT":                   "0",
	"DAILY_LEECH_LIMIT":            "0",
	"DAILY_TASK_LIMIT":             "0",
	"AUTO_DELETE_MESSAGE_DURATION": "60",
	"SHORTENER":                    "",
	"SHORTENER_API":                "",
	"SHORTENER_ENABLED":            "true",
	"TOKEN_TIMEOUT":                "0",
	"UPSTREAM_REPO":                "https://github.com/aquib4040/azml-go",
	"UPSTREAM_BRANCH":              "main",
}

func isBoolVar(key string) bool {
	switch key {
	case "AS_DOCUMENT", "AUTO_THUMBNAIL", "EQUAL_SPLITS", "MEDIA_GROUP", "SHORTENER_ENABLED":
		return true
	default:
		return false
	}
}

func maskAPI(api string) string {
	if api == "" {
		return "Belum Diatur"
	}
	if len(api) > 8 {
		return api[:4] + "***" + api[len(api)-4:]
	}
	return "***"
}

func parseDurationSec(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0
	}
	if strings.HasSuffix(s, "d") {
		d, err := strconv.ParseInt(strings.TrimSuffix(s, "d"), 10, 64)
		if err == nil {
			return d * 86400
		}
	}
	if strings.HasSuffix(s, "h") {
		h, err := strconv.ParseInt(strings.TrimSuffix(s, "h"), 10, 64)
		if err == nil {
			return h * 3600
		}
	}
	if strings.HasSuffix(s, "m") {
		m, err := strconv.ParseInt(strings.TrimSuffix(s, "m"), 10, 64)
		if err == nil {
			return m * 60
		}
	}
	if strings.HasSuffix(s, "s") {
		s = strings.TrimSuffix(s, "s")
	}
	sec, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		return sec
	}
	return 0
}

func (m *BotSettingsModule) HandleBotSettings(ctx context.Context, peer tg.InputPeerClass, msg *types.MessageContext) {
	if !telegram.IsOwner(msg.UserID) && !telegram.IsSudo(msg.UserID, m.userMap) {
		_, _ = m.sender.SendMessage(ctx, peer, "Hanya Pemilik/Sudo yang dapat mengakses Pengaturan Bot!", msg.MessageID, nil)
		return
	}

	parts := strings.Fields(msg.Text)
	if len(parts) >= 3 {
		key := strings.ToUpper(parts[1])
		val := strings.Join(parts[2:], " ")
		if key == "USER_SESSION_STRING" && m.userSessionReloader != nil {
			uname, uid, isPrem, err := m.userSessionReloader(ctx, val)
			if err != nil {
				_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("× <b>Gagal mengatur USER_SESSION_STRING:</b> <code>%v</code>", err), msg.MessageID, nil)
				return
			}
			_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("✓ <b>USER_SESSION_STRING</b> berhasil diatur sebagai @%s (ID: %d)! Premium: %v", uname, uid, isPrem), msg.MessageID, nil)
			return
		}
		if key == "DUMP_CHAT_ID" {
			dID, _ := strconv.ParseInt(val, 10, 64)
			_ = config.UpdateConfigVar("DUMP_CHAT_ID", val)
			if m.uploader != nil {
				m.uploader.SetDumpChatID(dID)
			}
			if m.leechMod != nil {
				m.leechMod.SetDumpChatID(dID)
			}
			_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("✓ <b>DUMP_CHAT_ID</b> diatur ke: <code>%d</code>", dID), msg.MessageID, nil)
			return
		}
		if err := config.UpdateConfigVar(key, val); err == nil {
			_, _ = m.sender.SendMessage(ctx, peer, fmt.Sprintf("✓ <b>%s</b> telah diatur ke: <code>%s</code>", key, val), msg.MessageID, nil)
			return
		}
	}

	text, btn := m.buildMainMenu()
	_, _ = m.sender.SendMessageWithButtons(ctx, peer, text, msg.MessageID, btn, 1)
}

func (m *BotSettingsModule) buildMainMenu() (string, *telegram.ButtonMaker) {
	text := fmt.Sprintf("⌬ <b><u>%s</u> :</b>\n\n<i>Pilih kategori di bawah untuk melihat atau mengubah konfigurasi:</i>", telegram.ToSmallCaps("Pengaturan Bot"))
	btn := telegram.NewButtonMaker()
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Variabel Konfigurasi"), "botset var 0", "primary")
	btn.Row()
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Sesi Pengguna (4GB+)"), "botset session", "primary")
	btn.Row()
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Berkas Pribadi"), "botset private", "primary")
	btn.Row()
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Pengaturan Aria2c"), "botset aria", "primary")
	btn.Row()
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Pengaturan Pemendek Tautan"), "botset shortener", "primary")
	btn.Row()
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), "botset close", "danger")
	btn.Row()
	return text, btn
}

func (m *BotSettingsModule) buildSessionMenu() (string, *telegram.ButtonMaker) {
	cfg := config.Get()
	hasSession := cfg.UserSessionString != ""
	isPrem := config.IsPremiumUser
	maxSplit := config.GetMaxSplitSize()

	status := "Belum Dikonfigurasi (Batas Standar 2GB)"
	if hasSession {
		if isPrem {
			status = "Aktif — Premium (Unggah 4GB Diaktifkan)"
		} else {
			status = "Aktif — Non-Premium (Unggah 2GB)"
		}
	}

	dumpInfo := "Belum Diatur"
	if cfg.DumpChatID != 0 {
		dumpInfo = fmt.Sprintf("<code>%d</code>", cfg.DumpChatID)
	}

	text := fmt.Sprintf("⌬ <b><u>%s</u> :</b>\n\n"+
		"┠ <b>Status:</b> %s\n"+
		"┠ <b>Ukuran Pemisahan Maks:</b> <code>%s</code>\n"+
		"┠ <b>Kanal Dump:</b> %s\n"+
		"┠ <b>String Sesi:</b> <code>%s</code>\n"+
		"┖ <b>Alur Unggah:</b> <code>String Pengguna ➔ Kanal Dump ➔ DM Pengguna</code>\n\n"+
		"<i>Format string sesi yang didukung: Pyrogram (v1, v2), Pyrofork, Telethon, gotd</i>\n\n"+
		"<i>Klik <b>Atur String Sesi</b> untuk menyimpan/memperbarui sesi pengguna Anda bagi unggahan 4GB:</i>",
		telegram.ToSmallCaps("Pengaturan Sesi Pengguna"),
		status,
		ext.GetReadableFileSize(maxSplit),
		dumpInfo,
		config.MaskSession(cfg.UserSessionString),
	)

	btn := telegram.NewButtonMaker()
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur String Sesi"), "botset prompt_session", "primary")
	if hasSession {
		btn.IButtonStyled("• "+telegram.ToSmallCaps("Atur Ulang Sesi"), "botset resetsession", "danger")
	}
	btn.Row()
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur Kanal Dump"), "botset prompt_dump", "primary")
	btn.Row()
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), "botset back", "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), "botset close", "danger")
	btn.Row()

	return text, btn
}

func (m *BotSettingsModule) buildVarsMenu(page int) (string, *telegram.ButtonMaker) {
	cfgMap := config.GetConfigMap()
	keys := configVarKeys

	limit := 8
	totalPages := (len(keys) + limit - 1) / limit
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * limit
	end := start + limit
	if end > len(keys) {
		end = len(keys)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⌬ <b><u>%s</u></b> | <b>Halaman: %d/%d</b>\n\n", telegram.ToSmallCaps("Variabel Konfigurasi"), page+1, totalPages))

	btn := telegram.NewButtonMaker()
	for i := start; i < end; i++ {
		k := keys[i]
		v := cfgMap[k]
		if v == "" {
			v = "Belum Diatur"
		}
		sb.WriteString(fmt.Sprintf("┠ <b>%s:</b> <code>%s</code>\n", k, v))

		btn.IButtonStyled("◈ "+telegram.ToSmallCaps(k), fmt.Sprintf("botset editvar %s %d", k, page), "primary")
		if (i-start)%2 == 1 || i == end-1 {
			btn.Row()
		}
	}
	sb.WriteString("\n<i>Ketuk variabel untuk melihat, mengubah, atau menyunting nilainya secara langsung.</i>")

	// Page jump buttons (AZML style numbered jump bar, up to 6 per row)
	for p := 0; p < totalPages; p++ {
		if p == page {
			btn.IButtonStyled(fmt.Sprintf("« %d »", p+1), fmt.Sprintf("botset var %d", p), "success")
		} else {
			btn.IButtonStyled(fmt.Sprintf("%d", p+1), fmt.Sprintf("botset var %d", p), "primary")
		}
		if (p+1)%6 == 0 || p == totalPages-1 {
			btn.Row()
		}
	}

	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), "botset back", "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), "botset close", "danger")
	btn.Row()

	return sb.String(), btn
}

func (m *BotSettingsModule) buildVarDetailMenu(key string, page int, editing bool) (string, *telegram.ButtonMaker) {
	cfgMap := config.GetConfigMap()
	v, exists := cfgMap[key]
	if !exists || v == "" {
		v = "Belum Diatur"
	}

	btn := telegram.NewButtonMaker()

	if isBoolVar(key) {
		text := fmt.Sprintf("⌬ <b><u>%s</u> :</b>\n\n"+
			"┠ <b>Variabel:</b> <code>%s</code>\n"+
			"┖ <b>Nilai Saat Ini:</b> <code>%s</code>\n\n"+
			"<i>Pilih nilai yang valid untuk variabel di atas:</i>",
			telegram.ToSmallCaps("Konfigurasi Boolean"), key, v)

		btn.IButtonStyled("✓ "+telegram.ToSmallCaps("Benar"), fmt.Sprintf("botset boolvar %s true %d", key, page), "success")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Salah"), fmt.Sprintf("botset boolvar %s false %d", key, page), "danger")
		btn.Row()
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("botset var %d", page), "primary")
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), "botset close", "danger")
		btn.Row()
		return text, btn
	}

	if editing {
		text := fmt.Sprintf("⌬ <b><u>%s</u> :</b>\n\n"+
			"┠ <b>Variabel:</b> <code>%s</code>\n"+
			"┖ <b>Nilai Saat Ini:</b> <code>%s</code>\n\n"+
			"<i>Kirim nilai yang valid untuk variabel di atas di obrolan.</i> <b>Batas waktu:</b> 60 detik\n"+
			"<i>Kirim /cancel untuk membatalkan.</i>",
			telegram.ToSmallCaps("Sunting Variabel"), key, v)

		btn.IButtonStyled("× "+telegram.ToSmallCaps("Batalkan Penyuntingan"), fmt.Sprintf("botset editvar %s %d", key, page), "danger")
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("botset var %d", page), "primary")
		btn.Row()
		return text, btn
	}

	text := fmt.Sprintf("⌬ <b><u>%s</u> :</b>\n\n"+
		"┠ <b>Variabel:</b> <code>%s</code>\n"+
		"┖ <b>Nilai Saat Ini:</b> <code>%s</code>\n\n"+
		"<i>Klik <b>Sunting Nilai</b> untuk memperbarui pengaturan ini secara langsung:</i>",
		telegram.ToSmallCaps("Detail Variabel"), key, v)

	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Sunting Nilai"), fmt.Sprintf("botset editvar_prompt %s %d", key, page), "success")
	btn.IButtonStyled("• "+telegram.ToSmallCaps("Atur Ulang"), fmt.Sprintf("botset resetvar %s %d", key, page), "danger")
	btn.Row()
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), fmt.Sprintf("botset var %d", page), "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), "botset close", "danger")
	btn.Row()

	return text, btn
}

func (m *BotSettingsModule) HandleSessionValue(ctx context.Context, peer tg.InputPeerClass, promptMsgID int, key, val string, page int) {
	val = strings.TrimSpace(val)

	if key == "USER_SESSION_STRING" {
		if m.userSessionReloader != nil {
			uname, uid, isPrem, err := m.userSessionReloader(ctx, val)
			if err != nil {
				errMsg := fmt.Sprintf("\n\n× <b>Gagal menghubungkan Sesi Pengguna:</b> <code>%v</code>\n<i>Silakan periksa bahwa string sesi valid dan aktif.</i>", err)
				if page == -2 {
					text, btn := m.buildSessionMenu()
					text += errMsg
					if m.sender != nil {
						_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
					}
				} else {
					text, btn := m.buildVarDetailMenu(key, page, false)
					text += errMsg
					if m.sender != nil {
						_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
					}
				}
				return
			}

			successMsg := fmt.Sprintf("\n\n✓ <b>Sesi Pengguna terhubung!</b>\n┠ <b>Pengguna:</b> @%s (ID: %d)\n┠ <b>Premium:</b> %v\n┠ <b>Pemisahan Maks:</b> %s\n┖ <b>Alur Unggah:</b> <code>String Pengguna ➔ Kanal Dump ➔ DM Pengguna</code>",
				uname, uid, isPrem, ext.GetReadableFileSize(config.GetMaxSplitSize()))

			if page == -2 {
				text, btn := m.buildSessionMenu()
				text += successMsg
				if m.sender != nil {
					_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
				}
			} else {
				text, btn := m.buildVarDetailMenu(key, page, false)
				text += successMsg
				if m.sender != nil {
					_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
				}
			}
			return
		}
	}

	if key == "DUMP_CHAT_ID" {
		dID, _ := strconv.ParseInt(val, 10, 64)
		_ = config.UpdateConfigVar("DUMP_CHAT_ID", val)
		if m.uploader != nil {
			m.uploader.SetDumpChatID(dID)
		}
		if m.leechMod != nil {
			m.leechMod.SetDumpChatID(dID)
		}
		if page == -2 {
			text, btn := m.buildSessionMenu()
			text += fmt.Sprintf("\n\n✓ <b>Kanal Dump diatur ke:</b> <code>%d</code>", dID)
			if m.sender != nil {
				_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
			}
			return
		}
	}

	if key == "TOKEN_TIMEOUT" && page == -1 {
		sec := parseDurationSec(val)
		if sec > 0 {
			_ = config.UpdateConfigVar("TOKEN_TIMEOUT", strconv.FormatInt(sec, 10))
			_ = config.UpdateConfigVar("SHORTENER_ENABLED", "true")
		} else {
			_ = config.UpdateConfigVar("TOKEN_TIMEOUT", val)
		}
	} else {
		_ = config.UpdateConfigVar(key, val)
	}

	if page == -1 || key == "SHORTENER" || key == "SHORTENER_API" {
		text, btn := m.buildShortenerMenu()
		text += fmt.Sprintf("\n\n✓ <i>%s berhasil diperbarui!</i>", key)
		if m.sender != nil {
			_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
		}
		return
	}

	text, btn := m.buildVarDetailMenu(key, page, false)
	text += "\n\n✓ <i>Nilai berhasil diperbarui!</i>"
	if m.sender != nil {
		_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
	}
}

func (m *BotSettingsModule) HandleSessionCancelled(ctx context.Context, peer tg.InputPeerClass, promptMsgID int, key string, page int) {
	if page == -2 {
		text, btn := m.buildSessionMenu()
		text += "\n\n× <i>Penyuntingan dibatalkan.</i>"
		if m.sender != nil {
			_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
		}
		return
	}
	if page == -1 || key == "SHORTENER" || key == "SHORTENER_API" {
		text, btn := m.buildShortenerMenu()
		text += "\n\n× <i>Penyuntingan dibatalkan.</i>"
		if m.sender != nil {
			_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
		}
		return
	}
	text, btn := m.buildVarDetailMenu(key, page, false)
	text += "\n\n× <i>Penyuntingan dibatalkan.</i>"
	if m.sender != nil {
		_ = m.sender.EditMessageWithButtons(ctx, peer, promptMsgID, text, btn, 2)
	}
}

func (m *BotSettingsModule) buildAriaMenu() (string, *telegram.ButtonMaker) {
	text := "⌬ <b><u>Pengaturan Kinerja Aria2c</u>:</b>\n\n" +
		"┠ <b>max-connection-per-server:</b> <code>16</code> (Maks)\n" +
		"┠ <b>split:</b> <code>16</code> (Maks)\n" +
		"┠ <b>min-split-size:</b> <code>1M</code>\n" +
		"┠ <b>bt-max-peers:</b> <code>128</code>\n" +
		"┠ <b>disk-cache:</b> <code>64M</code>\n" +
		"┖ <b>allow-overwrite:</b> <code>true</code>\n\n" +
		"<i>Dioptimalkan untuk kinerja jaringan maksimal pada server VPS gratis.</i>"

	btn := telegram.NewButtonMaker()
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), "botset back", "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), "botset close", "danger")
	btn.Row()
	return text, btn
}

func (m *BotSettingsModule) buildPrivateMenu() (string, *telegram.ButtonMaker) {
	text := fmt.Sprintf("⌬ <b><u>%s:</u></b>\n\n"+
		"<i>Kirim salah satu berkas pribadi ini sebagai dokumen langsung ke bot:</i>\n\n"+
		"┠ <code>token.pickle</code> — token OAuth Google Drive (dimuat otomatis)\n"+
		"┠ <code>config.env</code> — variabel lingkungan bot (dimuat ulang otomatis)\n"+
		"┠ <code>cookies.txt</code> — cookie Netscape untuk yt-dlp & aria2c\n"+
		"┠ <code>shorteners.txt</code> — berkas perutean multi-pemendek tautan (dimuat ulang otomatis)\n"+
		"┖ <code>accounts.zip</code> — arsip akun layanan\n\n"+
		"<b>Pemrosesan Otomatis:</b>\n"+
		"Saat Anda membagikan <code>token.pickle</code> atau <code>shorteners.txt</code> ke bot, keduanya akan disimpan dan dimuat ulang segera secara otomatis.",
		telegram.ToSmallCaps("Berkas Pribadi"))

	btn := telegram.NewButtonMaker()
	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), "botset back", "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), "botset close", "danger")
	btn.Row()
	return text, btn
}

func (m *BotSettingsModule) HandleCallback(ctx context.Context, peer tg.InputPeerClass, queryID int64, msgID int, data string, fromUID int64) {
	if !telegram.IsOwner(fromUID) && !telegram.IsSudo(fromUID, m.userMap) {
		_ = m.sender.AnswerCallback(ctx, queryID, "Hanya Pemilik/Sudo yang dapat mengakses Pengaturan Bot!", true)
		return
	}

	parts := strings.Fields(data)
	if len(parts) < 2 {
		return
	}

	action := parts[1]
	switch action {
	case "close":
		ext.ClearPendingBotSettingSession(fromUID)
		_ = m.sender.DeleteMessage(ctx, peer, []int{msgID})
		_ = m.sender.AnswerCallback(ctx, queryID, "Ditutup", false)

	case "back":
		ext.ClearPendingBotSettingSession(fromUID)
		text, btn := m.buildMainMenu()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "var":
		ext.ClearPendingBotSettingSession(fromUID)
		page := 0
		if len(parts) >= 3 {
			page, _ = strconv.Atoi(parts[2])
		}
		text, btn := m.buildVarsMenu(page)
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "editvar":
		ext.ClearPendingBotSettingSession(fromUID)
		if len(parts) >= 3 {
			key := parts[2]
			page := 0
			if len(parts) >= 4 {
				page, _ = strconv.Atoi(parts[3])
			}
			text, btn := m.buildVarDetailMenu(key, page, false)
			_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
			_ = m.sender.AnswerCallback(ctx, queryID, "", false)
		}

	case "editvar_prompt":
		if len(parts) >= 3 {
			key := parts[2]
			page := 0
			if len(parts) >= 4 {
				page, _ = strconv.Atoi(parts[3])
			}
			ext.SetPendingBotSettingSession(fromUID, key, 0, msgID, page, 60*time.Second)
			text, btn := m.buildVarDetailMenu(key, page, true)
			_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
			_ = m.sender.AnswerCallback(ctx, queryID, "Kirim nilai baru dalam 60 detik...", false)
		}

	case "boolvar":
		if len(parts) >= 4 {
			key := parts[2]
			val := parts[3]
			page := 0
			if len(parts) >= 5 {
				page, _ = strconv.Atoi(parts[4])
			}
			_ = config.UpdateConfigVar(key, val)
			text, btn := m.buildVarDetailMenu(key, page, false)
			text += fmt.Sprintf("\n\n✓ <i>%s diatur ke %s!</i>", key, val)
			_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
			_ = m.sender.AnswerCallback(ctx, queryID, fmt.Sprintf("%s diatur ke %s", key, val), false)
		}

	case "resetvar":
		if len(parts) >= 3 {
			key := parts[2]
			page := 0
			if len(parts) >= 4 {
				page, _ = strconv.Atoi(parts[3])
			}
			defVal := defaultVarValues[key]
			if key == "USER_SESSION_STRING" && m.userSessionReloader != nil {
				_, _, _, _ = m.userSessionReloader(ctx, "")
			} else {
				_ = config.UpdateConfigVar(key, defVal)
			}
			text, btn := m.buildVarDetailMenu(key, page, false)
			text += fmt.Sprintf("\n\n• <i>%s diatur ulang ke bawaan: %s</i>", key, defVal)
			_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
			_ = m.sender.AnswerCallback(ctx, queryID, fmt.Sprintf("%s diatur ulang ke bawaan", key), false)
		}

	case "session":
		ext.ClearPendingBotSettingSession(fromUID)
		text, btn := m.buildSessionMenu()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "prompt_session":
		ext.SetPendingBotSettingSession(fromUID, "USER_SESSION_STRING", 0, msgID, -2, 60*time.Second)
		text := fmt.Sprintf("⌬ <b><u>%s</u> :</b>\n\n"+
			"Kirim string sesi Pyrogram (v1/v2), Pyrofork, Telethon, atau gotd Anda di obrolan.\n\n"+
			"<b>Batas waktu:</b> 60 detik\n<i>Kirim /cancel untuk membatalkan.</i>",
			telegram.ToSmallCaps("Atur String Sesi Pengguna"))
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Batalkan"), "botset session", "danger")
		btn.Row()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim string sesi di obrolan...", false)

	case "resetsession":
		ext.ClearPendingBotSettingSession(fromUID)
		if m.userSessionReloader != nil {
			_, _, _, _ = m.userSessionReloader(ctx, "")
		} else {
			_ = config.UpdateConfigVar("USER_SESSION_STRING", "")
			config.SetIsPremiumUser(false)
		}
		text, btn := m.buildSessionMenu()
		text += "\n\n• <i>Sesi Pengguna dihapus. Kembali ke batas standar 2GB.</i>"
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Sesi pengguna diatur ulang", false)

	case "prompt_dump":
		ext.SetPendingBotSettingSession(fromUID, "DUMP_CHAT_ID", 0, msgID, -2, 60*time.Second)
		text := fmt.Sprintf("⌬ <b><u>%s</u> :</b>\n\n"+
			"Kirim ID Kanal Dump (mis. <code>-100xxxxxxxxxx</code>) tempat berkas 2GB+ akan diunggah sebelum diteruskan ke DM Pengguna.\n\n"+
			"<b>Batas waktu:</b> 60 detik\n<i>Kirim /cancel untuk membatalkan.</i>",
			telegram.ToSmallCaps("Atur Kanal Dump"))
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Batalkan"), "botset session", "danger")
		btn.Row()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 1)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim ID Kanal Dump di obrolan...", false)

	case "aria":
		text, btn := m.buildAriaMenu()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "private":
		text, btn := m.buildPrivateMenu()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "shortener":
		text, btn := m.buildShortenerMenu()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "", false)

	case "toggleshrt":
		cfg := config.Get()
		if cfg.ShortenerEnabled && cfg.TokenTimeout > 0 {
			_ = config.UpdateConfigVar("SHORTENER_ENABLED", "false")
			_ = m.sender.AnswerCallback(ctx, queryID, "Pemendek Tautan Dinonaktifkan!", false)
		} else {
			_ = config.UpdateConfigVar("SHORTENER_ENABLED", "true")
			if cfg.TokenTimeout <= 0 {
				_ = config.UpdateConfigVar("TOKEN_TIMEOUT", "86400")
			}
			_ = m.sender.AnswerCallback(ctx, queryID, "Pemendek Tautan Diaktifkan (24 Jam)!", false)
		}
		text, btn := m.buildShortenerMenu()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)

	case "setshrt":
		if len(parts) >= 3 {
			sec := parts[2]
			_ = config.UpdateConfigVar("TOKEN_TIMEOUT", sec)
			_ = config.UpdateConfigVar("SHORTENER_ENABLED", "true")
			valInt, _ := strconv.ParseInt(sec, 10, 64)
			_ = m.sender.AnswerCallback(ctx, queryID, fmt.Sprintf("Masa berlaku diatur ke %s", ext.GetReadableTime(valInt)), false)
		}
		text, btn := m.buildShortenerMenu()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, text, btn, 2)

	case "prompt_shrt_domain":
		cfg := config.Get()
		cur := cfg.Shortener
		if cur == "" {
			cur = "Belum Diatur"
		}
		ext.SetPendingBotSettingSession(fromUID, "SHORTENER", 0, msgID, -1, 60*time.Second)
		promptText := fmt.Sprintf("⌬ <b><u>%s</u> :</b>\n\n"+
			"┠ <b>Domain Saat Ini:</b> <code>%s</code>\n\n"+
			"<i>Kirim domain pemendek tautan (mis. <code>droplink.co</code> atau <code>https://api.shareus.io/short?api=%%s&url=%%s</code>).</i>\n"+
			"<b>Batas waktu:</b> 60 detik\n"+
			"<i>Kirim /cancel untuk membatalkan.</i>",
			telegram.ToSmallCaps("Atur Domain Pemendek Tautan"), cur)
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Batalkan"), "botset shortener", "danger")
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), "botset shortener", "primary")
		btn.Row()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, promptText, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim domain pemendek tautan dalam 60 detik...", false)

	case "prompt_shrt_api":
		cfg := config.Get()
		cur := maskAPI(cfg.ShortenerAPI)
		ext.SetPendingBotSettingSession(fromUID, "SHORTENER_API", 0, msgID, -1, 60*time.Second)
		promptText := fmt.Sprintf("⌬ <b><u>%s</u> :</b>\n\n"+
			"┠ <b>Kunci API Saat Ini:</b> <code>%s</code>\n\n"+
			"<i>Kirim kunci API pemendek tautan Anda di obrolan.</i>\n"+
			"<b>Batas waktu:</b> 60 detik\n"+
			"<i>Kirim /cancel untuk membatalkan.</i>",
			telegram.ToSmallCaps("Atur Kunci API Pemendek Tautan"), cur)
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Batalkan"), "botset shortener", "danger")
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), "botset shortener", "primary")
		btn.Row()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, promptText, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim kunci API pemendek tautan dalam 60 detik...", false)

	case "prompt_shrt_val":
		cfg := config.Get()
		cur := ext.GetReadableTime(cfg.TokenTimeout)
		ext.SetPendingBotSettingSession(fromUID, "TOKEN_TIMEOUT", 0, msgID, -1, 60*time.Second)
		promptText := fmt.Sprintf("⌬ <b><u>%s</u> :</b>\n\n"+
			"┠ <b>Masa Berlaku Saat Ini:</b> <code>%s</code>\n\n"+
			"<i>Kirim durasi masa berlaku kustom dalam detik (mis. <code>86400</code>) atau dengan akhiran (mis. <code>24h</code>, <code>12h</code>, <code>2d</code>).</i>\n"+
			"<b>Batas waktu:</b> 60 detik\n"+
			"<i>Kirim /cancel untuk membatalkan.</i>",
			telegram.ToSmallCaps("Masa Berlaku Token Kustom"), cur)
		btn := telegram.NewButtonMaker()
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Batalkan"), "botset shortener", "danger")
		btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), "botset shortener", "primary")
		btn.Row()
		_ = m.sender.EditMessageWithButtons(ctx, peer, msgID, promptText, btn, 2)
		_ = m.sender.AnswerCallback(ctx, queryID, "Kirim durasi masa berlaku dalam 60 detik...", false)
	}
}

func (m *BotSettingsModule) buildShortenerMenu() (string, *telegram.ButtonMaker) {
	cfg := config.Get()
	isActive := cfg.ShortenerEnabled && cfg.TokenTimeout > 0
	valStr := ext.GetReadableTime(cfg.TokenTimeout)
	if !isActive {
		valStr = "Nonaktif (Tidak Ada)"
	}

	stEmoji := "Nonaktif"
	if isActive {
		stEmoji = "Aktif"
	}

	shDomain := cfg.Shortener
	if shDomain == "" {
		shDomain = "Belum Diatur"
	}
	shAPI := maskAPI(cfg.ShortenerAPI)

	shorteners := ext.GetShorteners()
	shrtsCount := len(shorteners)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⌬ <b><u>%s :</u></b>\n\n"+
		"┎ <b>Akses Pemendek Tautan :</b> <code>%s</code>\n"+
		"┠ <b>Masa Berlaku Token :</b> <code>%s</code>\n"+
		"┠ <b>Domain Bawaan :</b> <code>%s</code>\n"+
		"┖ <b>Kunci API Bawaan :</b> <code>%s</code>\n\n",
		telegram.ToSmallCaps("Pengaturan Pemendek Tautan"), stEmoji, valStr, shDomain, shAPI))

	if shrtsCount > 0 {
		sb.WriteString(fmt.Sprintf("➲ <b><u>%s (%d) :</u></b>\n", telegram.ToSmallCaps("Multi Pemendek Tautan"), shrtsCount))
		for i, sh := range shorteners {
			if i >= 5 {
				sb.WriteString(fmt.Sprintf("• <i>... dan %d lagi di shorteners.txt</i>\n", shrtsCount-5))
				break
			}
			masked := maskAPI(sh.APIKey)
			sb.WriteString(fmt.Sprintf("• <code>%d. %s</code> | <i>Kunci:</i> <code>%s</code>\n", i+1, sh.Domain, masked))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("➲ <b>Deskripsi:</b> <i>Saat diaktifkan, pengguna harus membuat token sementara melalui pemendek tautan sebelum melakukan leech.</i>")

	btn := telegram.NewButtonMaker()
	if isActive {
		btn.IButtonStyled("× "+telegram.ToSmallCaps("Nonaktifkan Pemendek Tautan"), "botset toggleshrt", "danger")
	} else {
		btn.IButtonStyled("✓ "+telegram.ToSmallCaps("Aktifkan Pemendek Tautan (24 jam)"), "botset toggleshrt", "success")
	}
	btn.Row()

	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("24 Jam"), "botset setshrt 86400", "primary")
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("12 Jam"), "botset setshrt 43200", "primary")
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("6 Jam"), "botset setshrt 21600", "primary")
	btn.Row()

	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur Domain"), "botset prompt_shrt_domain", "primary")
	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Atur Kunci API"), "botset prompt_shrt_api", "primary")
	btn.Row()

	btn.IButtonStyled("◈ "+telegram.ToSmallCaps("Masa Berlaku Kustom"), "botset prompt_shrt_val", "primary")
	btn.Row()

	btn.IButtonStyled("« "+telegram.ToSmallCaps("Kembali"), "botset back", "primary")
	btn.IButtonStyled("× "+telegram.ToSmallCaps("Tutup"), "botset close", "danger")
	btn.Row()

	return sb.String(), btn
}
