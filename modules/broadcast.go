package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"azml-go/helper/ext"
	"azml-go/helper/telegram"
	"azml-go/types"

	"github.com/gotd/td/tg"
)

type BroadcastModule struct {
	sender  *telegram.MessageSender
	userMap *types.SafeUserMap
}

func NewBroadcastModule(sender *telegram.MessageSender, userMap *types.SafeUserMap) *BroadcastModule {
	return &BroadcastModule{
		sender:  sender,
		userMap: userMap,
	}
}

func (m *BroadcastModule) HandleBroadcast(ctx context.Context, peer tg.InputPeerClass, msg *types.MessageContext) {
	if !telegram.IsOwner(msg.UserID) && !telegram.IsSudo(msg.UserID, m.userMap) {
		_, _ = m.sender.SendMessage(ctx, peer, "Hanya Pemilik dan Sudo yang dapat menyiarkan pesan!", msg.MessageID, nil)
		return
	}

	replyMsg := msg.Text
	hasReply := false
	if msg.IsReply && msg.ReplyText != "" {
		replyMsg = msg.ReplyText
		hasReply = true
	} else {
		parts := strings.Fields(msg.Text)
		if len(parts) > 1 {
			replyMsg = strings.Join(parts[1:], " ")
		} else {
			helpMsg := "<b>Penggunaan Siaran:</b>\n" +
				"• Balas pesan apa pun: <code>/broadcast [-q]</code>\n" +
				"• Kirim teks: <code>/broadcast [Your message here]</code>\n\n" +
				"<b>Opsi:</b>\n" +
				"<code>-q</code> Kirim secara senyap (tanpa notifikasi)"
			_, _ = m.sender.SendMessage(ctx, peer, helpMsg, msg.MessageID, nil)
			return
		}
	}

	quiet := strings.Contains(msg.Text, "-q")

	users := m.userMap.GetAllUsers()
	if len(users) == 0 {
		_, _ = m.sender.SendMessage(ctx, peer, "Tidak ada pengguna ditemukan di database untuk menerima siaran.", msg.MessageID, nil)
		return
	}

	start := time.Now()
	total := len(users)
	success := 0
	failed := 0

	statID, _ := m.sender.SendMessage(ctx, peer, "<i>Memulai siaran...</i>", msg.MessageID, nil)

	go func() {
		for _, u := range users {
			targetPeer := &tg.InputPeerUser{UserID: u.UserID}
			var sendErr error
			if hasReply {
				_, sendErr = m.sender.SendMessage(ctx, targetPeer, replyMsg, 0, nil)
			} else {
				_, sendErr = m.sender.SendMessage(ctx, targetPeer, replyMsg, 0, nil)
			}

			if sendErr != nil {
				failed++
			} else {
				success++
			}

			// Gentle throttle to avoid flood wait
			time.Sleep(100 * time.Millisecond)
		}

		resultText := fmt.Sprintf("⌬ <b><i>Siaran Selesai:</i></b>\n"+
			"┠ <b>Total Pengguna:</b> <code>%d</code>\n"+
			"┠ <b>Berhasil:</b> <code>%d</code>\n"+
			"┠ <b>Gagal:</b> <code>%d</code>\n"+
			"┖ <b>Durasi:</b> <code>%s</code>",
			total, success, failed, ext.GetReadableTime(int64(time.Since(start).Seconds())))
		if quiet {
			resultText += "\n<i>Dikirim secara senyap (-q)</i>"
		}

		_ = m.sender.EditMessage(ctx, peer, statID, resultText, nil)
	}()
}
