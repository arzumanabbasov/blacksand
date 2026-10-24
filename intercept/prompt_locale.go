package intercept

import "github.com/arzumanabbasov/blacksand/locale"

func init() {
	locale.Register(JudgeContextBoundary, JudgeContextBoundary)
	locale.Register(JudgeOutputContract, JudgeOutputContract)
	locale.Register(DefaultJudgePrompt, DefaultJudgePrompt)
}
