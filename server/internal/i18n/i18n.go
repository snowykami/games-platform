package i18n

import (
	"net/http"
	"strings"
)

type Locale string

const (
	LocaleEN Locale = "en"
	LocaleZH Locale = "zh"
)

var messages = map[Locale]map[string]string{
	LocaleEN: {
		"admin_cannot_be_banned":    "Admin users cannot be banned",
		"admin_required":            "Admin role required",
		"ai_only_lobby":             "AI can only be added in the lobby",
		"ai_player_not_found":       "AI player not found",
		"card_not_found":            "Card not found",
		"card_not_playable":         "Card is not playable",
		"failed_encode_response":    "Failed to encode response",
		"frontend_index_missing":    "Frontend index.html not found",
		"game_already_started":      "Game already started",
		"game_not_playing":          "Game is not playing",
		"invalid_ai_payload":        "Invalid AI payload",
		"invalid_catch_uno_payload": "Invalid UNO catch payload",
		"invalid_claims":            "OIDC claims are invalid",
		"invalid_code_exchange":     "OIDC code exchange failed",
		"invalid_display_name":      "Display name cannot be empty",
		"invalid_guest_uuid":        "Invalid guest UUID",
		"invalid_id_token":          "OIDC ID token is invalid",
		"invalid_json_body":         "Invalid JSON body",
		"invalid_message":           "Invalid message",
		"invalid_move_payload":      "Invalid move payload",
		"invalid_oidc_identity":     "Invalid OIDC identity",
		"invalid_oidc_state":        "Invalid OIDC state",
		"invalid_player_payload":    "Invalid player payload",
		"invalid_place_payload":     "Invalid place payload",
		"invalid_play_payload":      "Invalid play payload",
		"invalid_position":          "Position is outside the board",
		"invalid_speech":            "Speech cannot be empty",
		"invalid_speech_payload":    "Invalid speech payload",
		"invalid_turn":              "Invalid turn",
		"llm_illegal_action":        "LLM selected an illegal action",
		"llm_not_configured":        "LLM AI is not configured",
		"login_required":            "Login required",
		"no_legal_actions":          "No legal actions",
		"not_current_turn":          "Not your turn",
		"not_in_room":               "You are not in this room",
		"oidc_id_token_missing":     "OIDC ID token is missing",
		"oidc_provider_not_found":   "OIDC provider not found",
		"cannot_remove_host":        "The host cannot be removed",
		"only_host_add_ai":          "Only the host can add AI",
		"only_host_remove_player":   "Only the host can remove players",
		"only_host_start":           "Only the host can start",
		"piece_not_found":           "Piece not found",
		"piece_wrong_side":          "Piece does not belong to current player",
		"player_not_found":          "Player not found",
		"position_occupied":         "Position is occupied",
		"room_full":                 "Room is full",
		"room_not_found":            "Room not found",
		"remove_player_only_lobby":  "Players can only be removed in the lobby",
		"unknown_message_type":      "Unknown message type",
		"uno_call_unavailable":      "UNO call is not available",
		"uno_catch_unavailable":     "UNO catch is not available",
		"user_banned":               "User is banned",
		"user_not_found":            "User not found",
		"wild_color_required":       "Wild color is required",
		"xiangqi_illegal_move":      "Illegal move",
		"need_two_players":          "Need two players",
	},
	LocaleZH: {
		"admin_cannot_be_banned":    "管理员不能被封禁",
		"admin_required":            "需要平台管理员权限",
		"ai_only_lobby":             "AI 只能在准备页添加",
		"ai_player_not_found":       "AI 玩家不存在",
		"card_not_found":            "卡牌不存在",
		"card_not_playable":         "这张牌不能打出",
		"failed_encode_response":    "响应编码失败",
		"frontend_index_missing":    "前端 index.html 不存在",
		"game_already_started":      "游戏已经开始",
		"game_not_playing":          "游戏未在进行中",
		"invalid_ai_payload":        "AI 消息格式无效",
		"invalid_catch_uno_payload": "UNO 抓罚消息格式无效",
		"invalid_claims":            "OIDC claims 无效",
		"invalid_code_exchange":     "OIDC code 交换失败",
		"invalid_display_name":      "对局昵称不能为空",
		"invalid_guest_uuid":        "游客 UUID 无效",
		"invalid_id_token":          "OIDC id_token 无效",
		"invalid_json_body":         "JSON 请求体无效",
		"invalid_message":           "消息格式无效",
		"invalid_move_payload":      "移动消息格式无效",
		"invalid_oidc_identity":     "OIDC 身份无效",
		"invalid_oidc_state":        "OIDC state 无效",
		"invalid_player_payload":    "玩家消息格式无效",
		"invalid_place_payload":     "落子消息格式无效",
		"invalid_play_payload":      "出牌消息格式无效",
		"invalid_position":          "位置在棋盘外",
		"invalid_speech":            "发言不能为空",
		"invalid_speech_payload":    "发言消息格式无效",
		"invalid_turn":              "回合状态无效",
		"llm_illegal_action":        "LLM 选择了非法动作",
		"llm_not_configured":        "LLM AI 尚未配置",
		"login_required":            "需要登录",
		"no_legal_actions":          "没有合法动作",
		"not_current_turn":          "还没轮到你",
		"not_in_room":               "你不在这个房间中",
		"oidc_id_token_missing":     "缺少 OIDC id_token",
		"oidc_provider_not_found":   "OIDC provider 不存在",
		"cannot_remove_host":        "不能移除房主",
		"only_host_add_ai":          "只有房主可以添加 AI",
		"only_host_remove_player":   "只有房主可以移除玩家",
		"only_host_start":           "只有房主可以开始游戏",
		"piece_not_found":           "棋子不存在",
		"piece_wrong_side":          "不能移动非己方棋子",
		"player_not_found":          "玩家不存在",
		"position_occupied":         "位置已被占用",
		"room_full":                 "房间已满",
		"room_not_found":            "房间不存在",
		"remove_player_only_lobby":  "只能在准备页移除玩家",
		"unknown_message_type":      "未知消息类型",
		"uno_call_unavailable":      "当前不能喊 UNO",
		"uno_catch_unavailable":     "当前不能抓罚 UNO",
		"user_banned":               "用户已被封禁",
		"user_not_found":            "用户不存在",
		"wild_color_required":       "万能牌需要选择颜色",
		"xiangqi_illegal_move":      "非法走法",
		"need_two_players":          "需要两名玩家",
	},
}

func FromRequest(r *http.Request) Locale {
	return FromAcceptLanguage(r.Header.Get("Accept-Language"))
}

func FromAcceptLanguage(header string) Locale {
	header = strings.ToLower(header)
	if strings.Contains(header, "zh") {
		return LocaleZH
	}
	return LocaleEN
}

func T(locale Locale, key string) string {
	if value := messages[locale][key]; value != "" {
		return value
	}
	if value := messages[LocaleEN][key]; value != "" {
		return value
	}
	return key
}
