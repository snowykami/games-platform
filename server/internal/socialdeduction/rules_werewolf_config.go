package socialdeduction

import (
	"errors"
	"strings"
)

func applyDefaultWerewolfConfig(room *Room) {
	room.Werewolf.RoleConfig = defaultWerewolfConfig(len(room.Players))
	room.Werewolf.RolePresets = werewolfRolePresets(len(room.Players))
}

func reconcileWerewolfConfig(room *Room) {
	if room.Game != GameWerewolf || room.Phase != PhaseLobby {
		return
	}
	room.Werewolf.RolePresets = werewolfRolePresets(len(room.Players))
	if room.Werewolf.RoleConfig.Mode != "custom" {
		room.Werewolf.RoleConfig = defaultWerewolfConfig(len(room.Players))
		return
	}
	counts := room.Werewolf.RoleConfig.Counts
	counts.Villager += len(room.Players) - counts.total()
	if err := validateWerewolfCounts(counts, len(room.Players)); err != nil {
		room.Werewolf.RoleConfig = defaultWerewolfConfig(len(room.Players))
		return
	}
	room.Werewolf.RoleConfig.Counts = counts
	room.Werewolf.RoleConfig.Name = "自定义角色组"
}

func defaultWerewolfConfig(players int) WerewolfRoleConfig {
	presets := werewolfRolePresets(players)
	if len(presets) > 0 {
		preset := presets[0]
		return WerewolfRoleConfig{
			Mode:        "preset",
			PresetID:    preset.ID,
			Name:        preset.Name,
			Description: preset.Description,
			Counts:      preset.Counts,
		}
	}
	counts := fallbackWerewolfCounts(players)
	return WerewolfRoleConfig{
		Mode:        "custom",
		Name:        "自定义角色组",
		Description: "人数未满足开局时的临时角色配置。",
		Counts:      counts,
	}
}

func normalizeWerewolfConfig(config WerewolfRoleConfig, players int) (WerewolfRoleConfig, error) {
	mode := strings.TrimSpace(config.Mode)
	if mode == "" && strings.TrimSpace(config.PresetID) != "" {
		mode = "preset"
	}
	if mode != "custom" {
		for _, preset := range werewolfRolePresets(players) {
			if preset.ID == config.PresetID {
				return WerewolfRoleConfig{
					Mode:        "preset",
					PresetID:    preset.ID,
					Name:        preset.Name,
					Description: preset.Description,
					Counts:      preset.Counts,
				}, nil
			}
		}
		return WerewolfRoleConfig{}, errors.New("invalid_role_preset")
	}

	counts := config.Counts
	if err := validateWerewolfCounts(counts, players); err != nil {
		return WerewolfRoleConfig{}, err
	}
	return WerewolfRoleConfig{
		Mode:        "custom",
		Name:        "自定义角色组",
		Description: "由房主自由调整狼人、村民和神职数量。",
		Counts:      counts,
	}, nil
}

func werewolfRolePresets(players int) []WerewolfRolePreset {
	presets := map[int][]WerewolfRolePreset{
		6: {
			werewolfPreset("wwf-6-classic", "6人标准", "2 狼、1 预言家、1 女巫、2 村民。节奏直接，适合快速局。", 6, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Villager: 2}),
			werewolfPreset("wwf-6-guarded", "6人守卫局", "1 狼、1 预言家、1 女巫、1 守卫、2 村民。信息更安全。", 6, WerewolfRoleCounts{Werewolf: 1, Seer: 1, Witch: 1, Guard: 1, Villager: 2}),
			werewolfPreset("wwf-6-hunter", "6人猎人局", "2 狼、1 预言家、1 猎人、2 村民。死亡反击更刺激。", 6, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Hunter: 1, Villager: 2}),
		},
		7: {
			werewolfPreset("wwf-7-classic", "7人标准", "2 狼、1 预言家、1 女巫、1 猎人、2 村民。攻防均衡。", 7, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Hunter: 1, Villager: 2}),
			werewolfPreset("wwf-7-guarded", "7人守卫", "2 狼、1 预言家、1 女巫、1 守卫、2 村民。夜晚博弈更多。", 7, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Guard: 1, Villager: 2}),
			werewolfPreset("wwf-7-idiot", "7人白痴局", "2 狼、1 预言家、1 女巫、1 白痴、2 村民。放逐风险更高。", 7, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Idiot: 1, Villager: 2}),
		},
		8: {
			werewolfPreset("wwf-8-classic", "8人标准", "2 狼、预言家、女巫、猎人、白痴、2 村民。经典小板子。", 8, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 2}),
			werewolfPreset("wwf-8-guarded", "8人守卫", "2 狼、预言家、女巫、猎人、守卫、2 村民。夜晚更保守。", 8, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Hunter: 1, Guard: 1, Villager: 2}),
			werewolfPreset("wwf-8-wolfish", "8人狼压", "3 狼、预言家、女巫、猎人、2 村民。邪恶更强。", 8, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Villager: 2}),
		},
		9: {
			werewolfPreset("wwf-9-classic", "9人预女猎白", "3 狼、预言家、女巫、猎人、白痴、2 村民。发言强度高。", 9, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 2}),
			werewolfPreset("wwf-9-guarded", "9人守卫", "3 狼、预言家、女巫、猎人、守卫、2 村民。攻防均衡。", 9, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Guard: 1, Villager: 2}),
			werewolfPreset("wwf-9-balanced", "9人均衡", "2 狼、预言家、女巫、猎人、白痴、3 村民。好人容错更高。", 9, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 3}),
		},
		10: {
			werewolfPreset("wwf-10-classic", "10人预女猎白", "3 狼、预言家、女巫、猎人、白痴、3 村民。推荐配置。", 10, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 3}),
			werewolfPreset("wwf-10-guarded", "10人守卫", "3 狼、预言家、女巫、猎人、守卫、3 村民。防守变量更多。", 10, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Guard: 1, Villager: 3}),
			werewolfPreset("wwf-10-fullgod", "10人五神", "3 狼、预言家、女巫、猎人、白痴、守卫、2 村民。神职密度高。", 10, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Guard: 1, Villager: 2}),
		},
		11: {
			werewolfPreset("wwf-11-classic", "11人经典", "3 狼、预言家、女巫、猎人、白痴、守卫、3 村民。信息与狼刀平衡。", 11, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Guard: 1, Villager: 3}),
			werewolfPreset("wwf-11-wolfish", "11人狼压", "4 狼、预言家、女巫、猎人、白痴、3 村民。高压对抗。", 11, WerewolfRoleCounts{Werewolf: 4, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 3}),
			werewolfPreset("wwf-11-safe", "11人稳健", "3 狼、预言家、女巫、猎人、守卫、4 村民。适合轻松局。", 11, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Guard: 1, Villager: 4}),
		},
		12: {
			werewolfPreset("wwf-12-classic", "12人经典", "4 狼、预言家、女巫、猎人、白痴、守卫、3 村民。满桌推荐。", 12, WerewolfRoleCounts{Werewolf: 4, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Guard: 1, Villager: 3}),
			werewolfPreset("wwf-12-balanced", "12人均衡", "3 狼、预言家、女巫、猎人、白痴、守卫、4 村民。讨论时间更宽。", 12, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Guard: 1, Villager: 4}),
			werewolfPreset("wwf-12-wolfish", "12人狼压", "4 狼、预言家、女巫、猎人、白痴、4 村民。狼队进攻更直接。", 12, WerewolfRoleCounts{Werewolf: 4, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 4}),
		},
	}
	return append([]WerewolfRolePreset{}, presets[players]...)
}

func werewolfPreset(id string, name string, description string, players int, counts WerewolfRoleCounts) WerewolfRolePreset {
	return WerewolfRolePreset{ID: id, Name: name, Description: description, Players: players, Counts: counts}
}

func fallbackWerewolfCounts(players int) WerewolfRoleCounts {
	if players <= 0 {
		return WerewolfRoleCounts{}
	}
	counts := WerewolfRoleCounts{Villager: players}
	if players >= 2 {
		counts.Werewolf = 1
		counts.Villager--
	}
	if players >= 3 {
		counts.Seer = 1
		counts.Villager--
	}
	if players >= 4 {
		counts.Guard = 1
		counts.Villager--
	}
	if players >= 6 {
		counts.Witch = 1
		counts.Villager--
	}
	return counts
}

func expandWerewolfRoles(counts WerewolfRoleCounts) []Role {
	roles := []Role{}
	for range counts.Werewolf {
		roles = append(roles, RoleWerewolf)
	}
	for range counts.Seer {
		roles = append(roles, RoleSeer)
	}
	for range counts.Guard {
		roles = append(roles, RoleGuard)
	}
	for range counts.Witch {
		roles = append(roles, RoleWitch)
	}
	for range counts.Hunter {
		roles = append(roles, RoleHunter)
	}
	for range counts.Idiot {
		roles = append(roles, RoleIdiot)
	}
	for range counts.Villager {
		roles = append(roles, RoleVillager)
	}
	return roles
}

func validateWerewolfCounts(counts WerewolfRoleCounts, players int) error {
	if counts.Villager < 0 || counts.Werewolf < 0 || counts.Seer < 0 || counts.Guard < 0 || counts.Witch < 0 || counts.Hunter < 0 || counts.Idiot < 0 {
		return errors.New("invalid_role_count")
	}
	if counts.total() != players {
		return errors.New("role_count_mismatch")
	}
	if players >= werewolfMinPlayers && counts.Werewolf < 1 {
		return errors.New("need_werewolf")
	}
	if counts.Werewolf >= players {
		return errors.New("too_many_werewolves")
	}
	if counts.Seer > 1 || counts.Guard > 1 || counts.Witch > 1 || counts.Hunter > 1 || counts.Idiot > 1 {
		return errors.New("too_many_unique_roles")
	}
	return nil
}

func (counts WerewolfRoleCounts) total() int {
	return counts.Villager + counts.Werewolf + counts.Seer + counts.Guard + counts.Witch + counts.Hunter + counts.Idiot
}
