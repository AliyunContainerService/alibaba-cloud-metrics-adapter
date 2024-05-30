package costv2

type AccumulateOption string

const (
	AccumulateOptionNone    AccumulateOption = ""
	AccumulateOptionAll     AccumulateOption = "all"
	AccumulateOptionHour    AccumulateOption = "hour"
	AccumulateOptionDay     AccumulateOption = "day"
	AccumulateOptionWeek    AccumulateOption = "week"
	AccumulateOptionMonth   AccumulateOption = "month"
	AccumulateOptionQuarter AccumulateOption = "quarter"
)
