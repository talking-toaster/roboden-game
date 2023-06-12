package menus

import (
	"github.com/ebitenui/ebitenui/widget"
	"github.com/quasilyte/ge"
	"github.com/quasilyte/roboden-game/assets"
	"github.com/quasilyte/roboden-game/controls"
	"github.com/quasilyte/roboden-game/gameui/eui"
	"github.com/quasilyte/roboden-game/session"
)

type LeaderboardMenuController struct {
	state *session.State

	scene *ge.Scene
}

func NewLeaderboardMenuController(state *session.State) *LeaderboardMenuController {
	return &LeaderboardMenuController{state: state}
}

func (c *LeaderboardMenuController) Init(scene *ge.Scene) {
	c.scene = scene
	c.initUI()
}

func (c *LeaderboardMenuController) Update(delta float64) {
	if c.state.MainInput.ActionIsJustPressed(controls.ActionBack) {
		c.back()
		return
	}
}

func (c *LeaderboardMenuController) initUI() {
	addDemoBackground(c.state, c.scene)
	uiResources := c.state.Resources.UI

	root := eui.NewAnchorContainer()
	rowContainer := eui.NewRowLayoutContainer(10, nil)
	root.AddChild(rowContainer)

	d := c.scene.Dict()

	titleLabel := eui.NewCenteredLabel(d.Get("menu.main.leaderboard"), assets.BitmapFont3)
	rowContainer.AddChild(titleLabel)

	rowContainer.AddChild(eui.NewButton(uiResources, c.scene, d.Get("menu.leaderboard.classic"), func() {
		c.scene.Context().ChangeScene(NewLeaderboardLoadingController(c.state, "classic"))
	}))

	rowContainer.AddChild(eui.NewButton(uiResources, c.scene, d.Get("menu.leaderboard.arena"), func() {
		c.scene.Context().ChangeScene(NewLeaderboardLoadingController(c.state, "arena"))
	}))

	rowContainer.AddChild(eui.NewButton(uiResources, c.scene, d.Get("menu.leaderboard.inf_arena"), func() {
		c.scene.Context().ChangeScene(NewLeaderboardLoadingController(c.state, "inf_arena"))
	}))

	rowContainer.AddChild(eui.NewButton(uiResources, c.scene, d.Get("menu.leaderboard.reverse"), func() {
		c.scene.Context().ChangeScene(NewLeaderboardLoadingController(c.state, "reverse"))
	}))

	rowContainer.AddChild(eui.NewSeparator(widget.RowLayoutData{Stretch: true}))

	rowContainer.AddChild(eui.NewButton(uiResources, c.scene, d.Get("menu.back"), func() {
		c.back()
	}))

	uiObject := eui.NewSceneObject(root)
	c.scene.AddGraphics(uiObject)
	c.scene.AddObject(uiObject)
}

func (c *LeaderboardMenuController) back() {
	c.scene.Context().ChangeScene(NewMainMenuController(c.state))
}
