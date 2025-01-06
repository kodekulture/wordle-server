package service

import (
	"context"
	"fmt"

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

type Service struct {
	*gameService
	*hub
	r       random.RandomGen
	cr      repository.HubBackup
	wordGen word.Generator
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

func (s *Service) FinishGame(ctx context.Context, g *game.Game) error {
	s.DeleteRoom(g.ID)
	return s.gameService.FinishGame(ctx, g)
}

func (s *Service) CreateInvite(player game.Player, gameID uuid.UUID) string {
	return s.r.Store(player, gameID)
}

func (s *Service) GetInviteData(token string) (game.Player, uuid.UUID, bool) {
	return s.r.Get(token)
}

func (s *Service) loadHub(ctx context.Context) (map[uuid.UUID]*game.Room, error) {
	hub, err := s.cr.Load(func(g *game.Game) *game.Room {
		return game.NewRoom(g, s)
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "error loading hub")
	}
	return hub, nil
}

func (s *Service) dumpHub(_ context.Context, hub map[uuid.UUID]*game.Room) error {
	err := s.cr.Dump(hub)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "error storing hub")
	}
	return nil
}

func (s *Service) drop(_ context.Context) error {
	err := s.cr.Drop()
	if err != nil {
		log.Err(err).Caller().Msg("error dropping hub")
		return err
	}
	return nil
}

func (s *Service) Stop(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.dumpHub(ctx, s.rooms)
	if err != nil {
		log.Err(err).Caller().Msg("failed to dump hub")
	}
}

func New(appCtx context.Context, gr repository.Game, pr repository.Player, cr repository.HubBackup) (*Service, error) {
	s := &Service{
		r:           random.New(appCtx),
		gameService: newGameSrv(gr, pr),
		wordGen:     word.NewLocalGen(),
		cr:          cr,
	}
	data, err := s.loadHub(appCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to load hub: %s", err.Error())
	}
	s.hub = newHub(data)
	go s.drop(appCtx)
	return s, nil
}
