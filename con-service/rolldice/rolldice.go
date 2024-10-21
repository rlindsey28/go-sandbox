package rolldice

type DiceRoll struct {
	Rolls        int8           `json:"rolls"`
	Sides        int8           `json:"sides"`
	Distribution map[int8]int32 `json:"distribution"`
}
