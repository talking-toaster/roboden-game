package staging

import (
	"math"

	"github.com/quasilyte/ge"
	"github.com/quasilyte/ge/xslices"
	"github.com/quasilyte/gmath"
	"github.com/quasilyte/gsignal"
	"github.com/quasilyte/roboden-game/assets"
	"github.com/quasilyte/roboden-game/controls"
	"github.com/quasilyte/roboden-game/gameinput"
	"github.com/quasilyte/roboden-game/gameui"
)

const colonyVisionRadius float64 = 500.0

type humanPlayer struct {
	world     *worldState
	state     *playerState
	scene     *ge.Scene
	input     gameinput.Handler
	choiceGen *choiceGenerator

	choiceWindow         *choiceWindowNode
	rpanel               *rpanelNode
	cursor               *gameui.CursorNode
	radar                *radarNode
	screenButtons        *screenButtonsNode
	colonySelector       *ge.Sprite
	flyingColonySelector *ge.Sprite

	creepsState *creepsPlayerState

	EventExitPressed gsignal.Event[gsignal.Void]
}

type humanPlayerConfig struct {
	world       *worldState
	state       *playerState
	input       gameinput.Handler
	cursor      *gameui.CursorNode
	choiceGen   *choiceGenerator
	creepsState *creepsPlayerState
}

func newHumanPlayer(config humanPlayerConfig) *humanPlayer {
	p := &humanPlayer{
		world:       config.world,
		state:       config.state,
		scene:       config.world.rootScene,
		choiceGen:   config.choiceGen,
		input:       config.input,
		cursor:      config.cursor,
		creepsState: config.creepsState,
	}
	return p
}

func (p *humanPlayer) addColonyToCreepsRadar(colony *colonyCoreNode) {
	p.radar.AddColony(colony)
	colony.EventDestroyed.Connect(p, func(colony *colonyCoreNode) {
		p.radar.RemoveColony(colony)
	})
}

func (p *humanPlayer) Init() {
	p.state.Init(p.world)

	p.choiceWindow = newChoiceWindowNode(p.state.camera.Camera, p.world, p.input, p.cursor, p.creepsState != nil)
	p.world.nodeRunner.AddObject(p.choiceWindow)

	if p.world.config.InterfaceMode >= 2 {
		if p.creepsState != nil {
			p.rpanel = newCreepsRpanelNode(p.state.camera.Camera, p.creepsState)
		} else {
			p.rpanel = newRpanelNode(p.state.camera.Camera)
		}
		p.scene.AddObject(p.rpanel)
	}

	buttonsPos := gmath.Vec{X: 137, Y: 470}
	if p.world.config.EnemyBoss && p.world.config.InterfaceMode >= 1 {
		p.radar = newRadarNode(p.world, p, p.creepsState != nil)
		p.radar.Init(p.world.rootScene)
		if p.creepsState != nil {
			for _, c := range p.world.allColonies {
				p.addColonyToCreepsRadar(c)
			}
			p.world.EventColonyCreated.Connect(p, p.addColonyToCreepsRadar)
		}
	} else {
		buttonsPos = gmath.Vec{X: 8, Y: 470}
	}

	if len(p.world.cameras) == 1 {
		p.screenButtons = newScreenButtonsNode(p.state.camera.Camera, buttonsPos)
		p.screenButtons.Init(p.world.rootScene)
		p.screenButtons.EventToggleButtonPressed.Connect(p, p.onToggleButtonClicked)
		p.screenButtons.EventExitButtonPressed.Connect(p, p.onExitButtonClicked)
	}

	if p.creepsState != nil {
		p.state.camera.CenterOn(p.world.boss.pos)
	} else {
		p.colonySelector = p.scene.NewSprite(assets.ImageColonyCoreSelector)
		p.state.camera.Private.AddSpriteBelow(p.colonySelector)
		p.flyingColonySelector = p.scene.NewSprite(assets.ImageColonyCoreSelector)
		p.state.camera.Private.AddSpriteSlightlyAbove(p.flyingColonySelector)

		p.selectNextColony(true)
		p.state.camera.CenterOn(p.state.selectedColony.pos)
	}

	p.choiceGen.EventChoiceReady.Connect(p, p.choiceWindow.RevealChoices)
	p.choiceGen.EventChoiceSelected.Connect(p, func(choice selectedChoice) {
		p.choiceWindow.StartCharging(choice.Cooldown, choice.Index)
		if p.rpanel != nil && choice.Index != -1 {
			p.rpanel.UpdateMetrics()
		}
	})
}

func (p *humanPlayer) IsDisposed() bool { return false }

func (p *humanPlayer) GetState() *playerState { return p.state }

func (p *humanPlayer) Update(computedDelta, delta float64) {
	p.choiceWindow.Enabled = p.state.selectedColony != nil &&
		p.state.selectedColony.mode == colonyModeNormal

	if p.state.selectedColony != nil {
		flying := p.state.selectedColony.IsFlying()
		p.colonySelector.Visible = !flying
		p.flyingColonySelector.Visible = flying
	}

	if p.radar != nil {
		p.radar.Update(delta)
	}
}

func (p *humanPlayer) HandleInput() {
	selectedColony := p.state.selectedColony

	if p.input.ActionIsJustPressed(controls.ActionToggleColony) {
		p.onToggleButtonClicked(gsignal.Void{})
		return
	}

	if p.input.ActionIsJustPressed(controls.ActionToggleInterface) {
		p.state.camera.UI.Visible = !p.state.camera.UI.Visible
	}

	if selectedColony != nil || p.creepsState != nil {
		if cardIndex := p.choiceWindow.HandleInput(); cardIndex != -1 {
			if !p.choiceGen.TryExecute(cardIndex, gmath.Vec{}) {
				p.scene.Audio().PlaySound(assets.AudioError)
			}
			return
		}
	}

	handledClick := false
	clickPos, hasClick := p.cursor.ClickPos(controls.ActionClick)
	if len(p.state.colonies) > 1 {
		if hasClick {
			globalClickPos := p.state.camera.AbsPos(clickPos)
			selectDist := 40.0
			if p.world.deviceInfo.IsMobile {
				selectDist = 80.0
			}
			var closestColony *colonyCoreNode
			closestDist := math.MaxFloat64
			for _, colony := range p.state.colonies {
				if colony == p.state.selectedColony {
					continue
				}
				dist := colony.pos.DistanceTo(globalClickPos)
				if dist > selectDist {
					continue
				}
				if dist < closestDist {
					closestColony = colony
					closestDist = dist
				}
			}
			if closestColony != nil {
				p.selectColony(closestColony)
				handledClick = true
			}
		}
	}
	if handledClick {
		return
	}

	if selectedColony != nil && p.world.movementEnabled {
		if pos, ok := p.cursor.ClickPos(controls.ActionMoveChoice); ok {
			globalClickPos := p.state.camera.AbsPos(pos)
			if globalClickPos.DistanceTo(selectedColony.pos) > 28 {
				if !p.choiceGen.TryExecute(-1, globalClickPos) {
					p.scene.Audio().PlaySound(assets.AudioError)
				}
				return
			}
		}
	}

	if p.screenButtons != nil {
		p.screenButtons.HandleInput(clickPos)
	}
}

func (p *humanPlayer) selectNextColony(center bool) {
	colony := p.findNextColony()
	p.selectColony(colony)
	if center && p.state.selectedColony != nil {
		p.state.camera.ToggleCamera(p.state.selectedColony.pos)
	}
}

func (p *humanPlayer) findNextColony() *colonyCoreNode {
	if len(p.state.colonies) == 0 {
		return nil
	}
	if len(p.state.colonies) == 1 {
		return p.state.colonies[0]
	}
	index := xslices.Index(p.state.colonies, p.state.selectedColony)
	if index == len(p.state.colonies)-1 {
		index = 0
	} else {
		index++
	}
	return p.state.colonies[index]
}

func (p *humanPlayer) selectColony(colony *colonyCoreNode) {
	if p.state.selectedColony == colony {
		return
	}
	if p.state.selectedColony != nil {
		p.scene.Audio().PlaySound(assets.AudioBaseSelect)
		p.state.selectedColony.EventDestroyed.Disconnect(p)
		p.state.selectedColony.EventTeleported.Disconnect(p)
		if p.rpanel != nil {
			p.state.selectedColony.EventPrioritiesChanged.Disconnect(p)
		}
	}
	p.state.selectedColony = colony

	if p.radar != nil {
		p.radar.SetBase(p.state.selectedColony)
	}
	if p.rpanel != nil {
		p.rpanel.SetBase(p.state.selectedColony)
		p.rpanel.UpdateMetrics()
	}
	if p.state.selectedColony == nil {
		p.colonySelector.Visible = false
		p.flyingColonySelector.Visible = false
		return
	}
	p.state.selectedColony.EventDestroyed.Connect(p, func(_ *colonyCoreNode) {
		p.selectNextColony(false)
	})
	p.state.selectedColony.EventTeleported.Connect(p, func(colony *colonyCoreNode) {
		p.state.camera.ToggleCamera(colony.pos)
	})
	if p.rpanel != nil {
		p.state.selectedColony.EventPrioritiesChanged.Connect(p, func(_ *colonyCoreNode) {
			p.rpanel.UpdateMetrics()
		})
	}
	p.colonySelector.Pos.Base = &p.state.selectedColony.spritePos
	p.flyingColonySelector.Pos.Base = &p.state.selectedColony.spritePos
}

func (p *humanPlayer) onPanelUpdateRequested(gsignal.Void) {
	// FIXME: is it unused?
	p.rpanel.UpdateMetrics()
}

func (p *humanPlayer) onExitButtonClicked(gsignal.Void) {
	p.EventExitPressed.Emit(gsignal.Void{})
}

func (p *humanPlayer) onToggleButtonClicked(gsignal.Void) {
	if p.creepsState == nil {
		p.selectNextColony(true)
		return
	}

	if p.world.boss != nil {
		p.state.camera.ToggleCamera(p.world.boss.pos)
	}
}
