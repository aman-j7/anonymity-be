package game

import (
	"log"
	"math/rand"
	"sort"
	"time"
	"fmt"
	"anonymity/models"
	"anonymity/questions"
	"anonymity/appconstants"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

// HandleStartGame is called when the host sends "start_game". Lock: acquires room.Mu.
func (e *Engine) HandleStartGame(player *models.Player, room *models.Room) {
	room.Mu.Lock()
	defer room.Mu.Unlock()

	if !player.IsHost {
		SendError(player, "NOT_HOST", "Only the host can start the game")
		return
	}
	if room.Status != models.RoomStatusLobby {
		SendError(player, "INVALID_PHASE", "Game has already started")
		return
	}
	activeCount := room.ActivePlayerCount()

	if activeCount < appconstants.ActivePlayerCount {
		errorString := fmt.Sprintf("Need at least %d players to start",appconstants.ActivePlayerCount,)
		SendError(player, "MIN_PLAYERS", errorString)
		return
	}


	room.Status = models.RoomStatusPlaying

	qs := questions.GetDefaultQuestions()
	rand.Shuffle(len(qs), func(i, j int) { qs[i], qs[j] = qs[j], qs[i] })
	room.Questions = qs
	room.QuestionIdx = 0

	playerIDs := make([]string, 0, len(room.Players))
	for id, p := range room.Players {
		if p.Status == models.PlayerConnected {
			playerIDs = append(playerIDs, id)
		}
	}
	rand.Shuffle(len(playerIDs), func(i, j int) { playerIDs[i], playerIDs[j] = playerIDs[j], playerIDs[i] })
	room.FeaturedOrder = playerIDs
	room.FeaturedIdx = 0

	players := buildPlayerInfoList(room)

	BroadcastToRoom(room, "game_started", models.GameStartedPayload{
		TotalRounds: room.Settings.NumRounds,
		Players:     players,
	})

	e.startRound(room)
}

// startRound begins a new round. Must be called with room.Mu held.
func (e *Engine) startRound(room *models.Room) {
	if room.QuestionIdx >= len(room.Questions) {
		rand.Shuffle(len(room.Questions), func(i, j int) {
			room.Questions[i], room.Questions[j] = room.Questions[j], room.Questions[i]
		})
		room.QuestionIdx = 0
	}
	question := room.Questions[room.QuestionIdx]
	room.QuestionIdx++

	featuredPlayer := e.pickFeaturedPlayer(room)
	rendered := question.Render(featuredPlayer.Name)

	roundNum := len(room.Rounds) + 1
	now := time.Now()
	deadline := now.Add(time.Duration(room.Settings.AnswerTimeSec) * time.Second)

	round := &models.Round{
		RoundNumber:    roundNum,
		Question:       question,
		RenderedPrompt: rendered,
		FeaturedPlayer: featuredPlayer.ID,
		Phase:          models.PhaseAnswering,
		Answers:        make(map[string]*models.Answer),
		Votes:          make(map[string]*models.Vote),
		StartedAt:      now,
		PhaseDeadline:  deadline,
	}

	room.Rounds = append(room.Rounds, round)
	room.CurrentRound = len(room.Rounds) - 1

	BroadcastToRoom(room, "new_round", models.NewRoundPayload{
		RoundNumber: roundNum,
		TotalRounds: room.Settings.NumRounds,
		Question:    rendered,
		FeaturedPlayer: models.PlayerInfo{
			ID:   featuredPlayer.ID,
			Name: featuredPlayer.Name,
		},
		Deadline:     deadline.Format(time.RFC3339),
		TimeLimitSec: room.Settings.AnswerTimeSec,
	})

	if room.PhaseTimer != nil {
		room.PhaseTimer.Stop()
	}
	room.PhaseTimer = time.AfterFunc(time.Duration(room.Settings.AnswerTimeSec)*time.Second, func() {
		room.Mu.Lock()
		defer room.Mu.Unlock()
		log.Printf("Room %s: answer phase timed out for round %d", room.Code, roundNum)
		e.transitionToVoting(room)
	})
}

func (e *Engine) pickFeaturedPlayer(room *models.Room) *models.Player {
	for attempts := 0; attempts < len(room.FeaturedOrder)*2; attempts++ {
		idx := room.FeaturedIdx % len(room.FeaturedOrder)
		room.FeaturedIdx++
		pid := room.FeaturedOrder[idx]
		if p, ok := room.Players[pid]; ok && p.Status == models.PlayerConnected {
			return p
		}
	}
	for _, p := range room.Players {
		if p.Status == models.PlayerConnected {
			return p
		}
	}
	return nil
}

// HandleSubmitAnswer processes a player's answer. Lock: acquires room.Mu.
func (e *Engine) HandleSubmitAnswer(player *models.Player, room *models.Room, payload models.SubmitAnswerPayload) {
	room.Mu.Lock()
	defer room.Mu.Unlock()

	round := room.CurrentRoundData()
	if round == nil || round.Phase != models.PhaseAnswering {
		SendError(player, "INVALID_PHASE", "Not in answering phase")
		return
	}
	if _, exists := round.Answers[player.ID]; exists {
		SendError(player, "ALREADY_SUBMITTED", "You already submitted an answer")
		return
	}
	text := payload.Text
	if len(text) == 0 || len(text) > 300 {
		SendError(player, "INVALID_PAYLOAD", "Answer must be 1-300 characters")
		return
	}

	round.Answers[player.ID] = &models.Answer{
		PlayerID:    player.ID,
		PlayerName:  player.Name,
		Text:        text,
		SubmittedAt: time.Now(),
	}

	activeCount := room.ActivePlayerCount()
	allAnswered := len(round.Answers) >= activeCount

	BroadcastToRoom(room, "answer_submitted", models.AnswerSubmittedPayload{
		AnswersCount: len(round.Answers),
		TotalPlayers: activeCount,
		AllAnswered:  allAnswered,
	})

	if allAnswered {
		if room.PhaseTimer != nil {
			room.PhaseTimer.Stop()
		}
		e.transitionToVoting(room)
	}
}

// transitionToVoting moves from answering to voting. Must be called with room.Mu held.
func (e *Engine) transitionToVoting(room *models.Room) {
	round := room.CurrentRoundData()
	if round == nil || round.Phase != models.PhaseAnswering {
		return
	}

	round.Phase = models.PhaseVoting
	deadline := time.Now().Add(time.Duration(room.Settings.VoteTimeSec) * time.Second)
	round.PhaseDeadline = deadline

	answers := make([]models.VotingAnswer, 0, len(round.Answers))
	for pid, a := range round.Answers {
		answers = append(answers, models.VotingAnswer{
			PlayerID: pid,
			Text:     a.Text,
		})
	}
	rand.Shuffle(len(answers), func(i, j int) { answers[i], answers[j] = answers[j], answers[i] })

	BroadcastToRoom(room, "voting_start", models.VotingStartPayload{
		RoundNumber:  round.RoundNumber,
		Question:     round.RenderedPrompt,
		Answers:      answers,
		Deadline:     deadline.Format(time.RFC3339),
		TimeLimitSec: room.Settings.VoteTimeSec,
	})

	if room.PhaseTimer != nil {
		room.PhaseTimer.Stop()
	}
	room.PhaseTimer = time.AfterFunc(time.Duration(room.Settings.VoteTimeSec)*time.Second, func() {
		room.Mu.Lock()
		defer room.Mu.Unlock()
		log.Printf("Room %s: voting phase timed out for round %d", room.Code, round.RoundNumber)
		e.transitionToResults(room)
	})
}

// HandleSubmitVote processes a player's vote. Lock: acquires room.Mu.
func (e *Engine) HandleSubmitVote(player *models.Player, room *models.Room, payload models.SubmitVotePayload) {
	room.Mu.Lock()
	defer room.Mu.Unlock()

	round := room.CurrentRoundData()
	if round == nil || round.Phase != models.PhaseVoting {
		SendError(player, "INVALID_PHASE", "Not in voting phase")
		return
	}
	if _, exists := round.Votes[player.ID]; exists {
		SendError(player, "ALREADY_SUBMITTED", "You already voted")
		return
	}
	if payload.VotedForPlayerID == player.ID {
		SendError(player, "CANNOT_VOTE_SELF", "You cannot vote for your own answer")
		return
	}
	if _, exists := round.Answers[payload.VotedForPlayerID]; !exists {
		SendError(player, "INVALID_PLAYER", "Invalid answer selection")
		return
	}

	round.Votes[player.ID] = &models.Vote{
		VoterID:        player.ID,
		VotedForPlayer: payload.VotedForPlayerID,
		SubmittedAt:    time.Now(),
	}

	activeCount := room.ActivePlayerCount()
	allVoted := len(round.Votes) >= activeCount

	BroadcastToRoom(room, "vote_submitted", models.VoteSubmittedPayload{
		VotesCount:   len(round.Votes),
		TotalPlayers: activeCount,
		AllVoted:     allVoted,
	})

	if allVoted {
		if room.PhaseTimer != nil {
			room.PhaseTimer.Stop()
		}
		e.transitionToResults(room)
	}
}

// transitionToResults tallies votes, calculates scores, and broadcasts results.
// Must be called with room.Mu held.
func (e *Engine) transitionToResults(room *models.Room) {
	round := room.CurrentRoundData()
	if round == nil || round.Phase != models.PhaseVoting {
		return
	}

	round.Phase = models.PhaseResults

	for _, vote := range round.Votes {
		if answer, ok := round.Answers[vote.VotedForPlayer]; ok {
			answer.VoteCount++
		}
	}

	maxVotes := 0
	for _, a := range round.Answers {
		if a.VoteCount > maxVotes {
			maxVotes = a.VoteCount
		}
	}

	winnerIDs := make(map[string]bool)
	if maxVotes > 0 {
		for pid, a := range round.Answers {
			if a.VoteCount == maxVotes {
				a.IsWinner = true
				winnerIDs[pid] = true
			}
		}
	}

	roundScores := make(map[string]int)
	for pid, a := range round.Answers {
		points := a.VoteCount * 100
		if a.IsWinner {
			points += 200
		}
		roundScores[pid] = points
	}
	for _, vote := range round.Votes {
		if winnerIDs[vote.VotedForPlayer] {
			roundScores[vote.VoterID] += 50
		}
	}

	for pid, points := range roundScores {
		if p, ok := room.Players[pid]; ok {
			p.Score += points
		}
	}

	results := make([]models.AnswerResult, 0, len(round.Answers))
	for pid, a := range round.Answers {
		voters := make([]string, 0)
		for _, vote := range round.Votes {
			if vote.VotedForPlayer == pid {
				if voter, ok := room.Players[vote.VoterID]; ok {
					voters = append(voters, voter.Name)
				}
			}
		}
		results = append(results, models.AnswerResult{
			PlayerID:   pid,
			PlayerName: a.PlayerName,
			AnswerText: a.Text,
			VoteCount:  a.VoteCount,
			IsWinner:   a.IsWinner,
			Voters:     voters,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].VoteCount > results[j].VoteCount
	})

	scores := make([]models.PlayerScore, 0, len(room.Players))
	for _, p := range room.Players {
		scores = append(scores, models.PlayerScore{
			PlayerID:    p.ID,
			Name:        p.Name,
			RoundPoints: roundScores[p.ID],
			TotalScore:  p.Score,
		})
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	BroadcastToRoom(room, "round_results", models.RoundResultsPayload{
		RoundNumber:    round.RoundNumber,
		Question:       round.RenderedPrompt,
		Results:        results,
		Scores:         scores,
		NextRoundInSec: room.Settings.ResultTimeSec,
	})

	if room.PhaseTimer != nil {
		room.PhaseTimer.Stop()
	}
	room.PhaseTimer = time.AfterFunc(time.Duration(room.Settings.ResultTimeSec)*time.Second, func() {
		room.Mu.Lock()
		defer room.Mu.Unlock()
		e.transitionToNextRound(room)
	})
}

// transitionToNextRound starts the next round or ends the game.
// Must be called with room.Mu held.
func (e *Engine) transitionToNextRound(room *models.Room) {
	if room.Status != models.RoomStatusPlaying {
		return
	}
	if len(room.Rounds) < room.Settings.NumRounds {
		e.startRound(room)
	} else {
		e.endGame(room)
	}
}

// endGame finalizes the game and broadcasts the leaderboard.
// Must be called with room.Mu held.
func (e *Engine) endGame(room *models.Room) {
	room.Status = models.RoomStatusFinished
	if room.PhaseTimer != nil {
		room.PhaseTimer.Stop()
		room.PhaseTimer = nil
	}

	players := make([]*models.Player, 0, len(room.Players))
	for _, p := range room.Players {
		players = append(players, p)
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].Score > players[j].Score
	})

	leaderboard := make([]models.LeaderboardEntry, len(players))
	for i, p := range players {
		leaderboard[i] = models.LeaderboardEntry{
			Rank:       i + 1,
			PlayerID:   p.ID,
			Name:       p.Name,
			TotalScore: p.Score,
		}
	}

	roundsWon := make(map[string]int)
	for _, r := range room.Rounds {
		for pid, a := range r.Answers {
			if a.IsWinner {
				roundsWon[pid]++
			}
		}
	}

	mvp := models.MVPInfo{}
	if len(players) > 0 {
		mvp = models.MVPInfo{
			PlayerID:   players[0].ID,
			Name:       players[0].Name,
			TotalScore: players[0].Score,
			RoundsWon:  roundsWon[players[0].ID],
		}
	}

	BroadcastToRoom(room, "game_over", models.GameOverPayload{
		Leaderboard: leaderboard,
		MVP:         mvp,
	})
	log.Printf("Room %s: game over. Winner: %s with %d points", room.Code, mvp.Name, mvp.TotalScore)
}

// HandleUpdateSettings allows the host to change settings in the lobby.
// Lock: acquires room.Mu.
func (e *Engine) HandleUpdateSettings(player *models.Player, room *models.Room, payload models.UpdateSettingsPayload) {
	room.Mu.Lock()
	defer room.Mu.Unlock()

	if !player.IsHost {
		SendError(player, "NOT_HOST", "Only the host can update settings")
		return
	}
	if room.Status != models.RoomStatusLobby {
		SendError(player, "INVALID_PHASE", "Cannot change settings after game started")
		return
	}

	if payload.MaxPlayers != nil {
		v := *payload.MaxPlayers
		if v >= 3 && v <= 12 {
			room.Settings.MaxPlayers = v
		}
	}
	if payload.NumRounds != nil {
		v := *payload.NumRounds
		if v >= 1 && v <= 20 {
			room.Settings.NumRounds = v
		}
	}
	if payload.AnswerTimeSec != nil {
		v := *payload.AnswerTimeSec
		if v >= 10 && v <= 120 {
			room.Settings.AnswerTimeSec = v
		}
	}
	if payload.VoteTimeSec != nil {
		v := *payload.VoteTimeSec
		if v >= 10 && v <= 60 {
			room.Settings.VoteTimeSec = v
		}
	}

	BroadcastToRoom(room, "settings_updated", models.SettingsUpdatedPayload{
		Settings: room.Settings,
	})
}

// HandleKickPlayer removes a player from the lobby. Lock: acquires room.Mu.
func (e *Engine) HandleKickPlayer(player *models.Player, room *models.Room, payload models.KickPlayerPayload) {
	room.Mu.Lock()
	defer room.Mu.Unlock()

	if !player.IsHost {
		SendError(player, "NOT_HOST", "Only the host can kick players")
		return
	}
	if room.Status != models.RoomStatusLobby {
		SendError(player, "INVALID_PHASE", "Cannot kick players after game started")
		return
	}

	target, ok := room.Players[payload.PlayerID]
	if !ok {
		SendError(player, "INVALID_PLAYER", "Player not found")
		return
	}
	if target.ID == player.ID {
		SendError(player, "INVALID_PLAYER", "Cannot kick yourself")
		return
	}

	SendToPlayer(target, "kicked", models.KickedPayload{Reason: "Removed by host"})
	if target.Send != nil {
		close(target.Send)
		target.Send = nil
	}
	delete(room.Players, target.ID)

	BroadcastToRoom(room, "player_left", models.PlayerLeftPayload{
		PlayerID:    target.ID,
		PlayerName:  target.Name,
		Reason:      "kicked",
		PlayerCount: len(room.Players),
	})
}

// HandleEmojiReact broadcasts an emoji reaction. Lock: acquires room.Mu.
func (e *Engine) HandleEmojiReact(player *models.Player, room *models.Room, payload models.EmojiReactPayload) {
	room.Mu.Lock()
	defer room.Mu.Unlock()

	BroadcastToRoom(room, "emoji_reaction", models.EmojiReactionPayload{
		PlayerName: player.Name,
		Emoji:      payload.Emoji,
	})
}

// HandleDisconnect handles a player disconnecting. Lock: acquires room.Mu.
// connGen identifies which connection generation is disconnecting; if the player
// has since reconnected (newer connGen), we skip the disconnect entirely.
func (e *Engine) HandleDisconnect(player *models.Player, room *models.Room, connGen uint64) {
	room.Mu.Lock()
	defer room.Mu.Unlock()

	if player.ConnGen != connGen {
		return
	}

	if _, exists := room.Players[player.ID]; !exists {
		return
	}

	player.Status = models.PlayerDisconnected
	if player.Send != nil {
		close(player.Send)
		player.Send = nil
	}
	room.LastActivity = time.Now()

	BroadcastToRoom(room, "player_left", models.PlayerLeftPayload{
		PlayerID:    player.ID,
		PlayerName:  player.Name,
		Reason:      "disconnected",
		PlayerCount: room.ActivePlayerCount(),
	})

	if room.Status != models.RoomStatusPlaying {
		return
	}

	activeCount := room.ActivePlayerCount()
	if activeCount < 2 {
		log.Printf("Room %s: not enough players, ending game", room.Code)
		e.endGame(room)
		return
	}

	round := room.CurrentRoundData()
	if round == nil {
		return
	}

	switch round.Phase {
	case models.PhaseAnswering:
		if len(round.Answers) >= activeCount {
			if room.PhaseTimer != nil {
				room.PhaseTimer.Stop()
			}
			e.transitionToVoting(room)
		}
	case models.PhaseVoting:
		if len(round.Votes) >= activeCount {
			if room.PhaseTimer != nil {
				room.PhaseTimer.Stop()
			}
			e.transitionToResults(room)
		}
	}
}

func buildPlayerInfoList(room *models.Room) []models.PlayerInfo {
	players := make([]models.PlayerInfo, 0, len(room.Players))
	for _, p := range room.Players {
		players = append(players, models.PlayerInfo{
			ID:     p.ID,
			Name:   p.Name,
			Score:  p.Score,
			IsHost: p.IsHost,
			Status: string(p.Status),
		})
	}
	return players
}
