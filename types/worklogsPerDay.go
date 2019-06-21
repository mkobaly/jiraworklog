package types

//WorklogsPerDay is agg result data representing the number
//of hours worked per day
type WorklogsPerDay struct {
	Developer    string  `db:"developer"`
	Day          string  `db:"day"`
	TimeSpentHrs float64 `db:"timeSpentHrs"`
}

type WorklogGroupByChart struct {
	GroupBy      string  `db:"groupBy"`
	TimeSpentHrs float64 `db:"timeSpentHrs"`
}

type WorklogsAggQueryResult struct {
	Developer    string  `db:"developer"`
	Group        string  `db:"group"`
	TimeSpentHrs float64 `db:"timeSpentHrs"`
}

type WorklogsPerDevDay struct {
	Developer string  `db:"Developer"`
	Monday    float64 `db:"Monday"`
	Tuesday   float64 `db:"Tuesday"`
	Wednesday float64 `db:"Wednesday"`
	Thursday  float64 `db:"Thursday"`
	Friday    float64 `db:"Friday"`
}

type WorklogsPerDevWeek struct {
	Developer  string
	ThisWeek   float64
	LastWeek   float64
	TwoWeeks   float64
	ThreeWeeks float64
	FourWeeks  float64
}
