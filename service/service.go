package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/lordvidex/errs/v2"

	"github.com/rs/zerolog/log"

	"github.com/kodekulture/wordle-server/game"
	"github.com/kodekulture/wordle-server/game/word"
	"github.com/kodekulture/wordle-server/repository"
	"github.com/kodekulture/wordle-server/service/random"
)

var (
	ErrNoPlayer = errs.B().Code(errs.InvalidArgument).Msg("player not provided").Err()
)

// Service ...
type Service struct {
	*coldStorage
	*localStorage
	r       random.RandomGen
	wordGen word.Generator
	store   repository.Hub
}

// NewRoom creates a new room and returns the id of the game that is currently running in this room
func (s *Service) NewRoom(username string) string {
	wrd := s.wordGen.Generate(word.Length)
	log.Debug().Msg(wrd) // TODO: remove this on production, for now leave it for debugging
	g := game.New(username, word.New(wrd))
	room := game.NewRoom(g, s)
	s.SetRoom(g.ID, room)
	return room.ID()
}

// StartGame ...
func (s *Service) StartGame(ctx context.Context, g *game.Game) error {
	err := s.coldStorage.StartGame(ctx, g)
	if err != nil {
		return err
	}
	return s.store.CreateGame(ctx, g)
}

// WipeGameData ...
func (s *Service) WipeGameData(ctx context.Context, id uuid.UUID) error {
	err := s.coldStorage.WipeGameData(ctx, id)
	if err != nil {
		return err
	}
	return s.store.DeleteGame(ctx, id)
}

// GetRoom ...
func (s *Service) GetRoom(id uuid.UUID) (*game.Room, bool) {
	if r, ok := s.localStorage.GetRoom(id); ok {
		return r, ok
	}

	// does game exist in store?
	if !s.store.Exists(context.Background(), id) {
		return nil, false
	}

	// try to load game
	g, err := s.store.LoadGame(context.Background(), id)
	if err != nil {
		log.Error().Err(err).Str("source", "hub").Msg("failed to load game")
		return nil, false
	}
	// create and add room to hub
	r := game.NewRoom(g, s)
	s.SetRoom(g.ID, r)

	return r, true
}

// FinishGame ...
func (s *Service) FinishGame(ctx context.Context, g *game.Game) error {
	err := s.coldStorage.FinishGame(ctx, g)
	if err != nil {
		return err
	}
	s.DeleteRoom(g.ID)
	return s.store.DeleteGame(ctx, g.ID)
}

// ValidateWord ...
func (s *Service) ValidateWord(word string) bool {
	return s.wordGen.Validate(word)
}

func (s *Service) AddGuess(ctx context.Context, gameID uuid.UUID, player string, guess word.Word, isBest bool) error {
	return s.store.AddGuess(ctx, gameID, player, guess, isBest)
}

// CreateInvite ...
func (s *Service) CreateInvite(player game.Player, gameID uuid.UUID) string {
	return s.r.Store(player, gameID)
}

// GetInviteData ...
func (s *Service) GetInviteData(token string) (game.Player, uuid.UUID, bool) {
	return s.r.Get(token)
}

// New ...
func New(appCtx context.Context, gr repository.Game, pr repository.Player, h repository.Hub) *Service {
	return &Service{
		r:            random.New(appCtx),
		coldStorage:  newColdStorage(gr, pr),
		wordGen:      word.NewLocalGen(),
		localStorage: newLocalStorage(appCtx),
		store:        h,
	}
}
