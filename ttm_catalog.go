package main

type ttmCatalogItem struct {
	label       string
	description string
}

// ttmCatalog keeps the original resource filename as the stable runtime
// identifier while giving the Settings menu a human-readable title and the
// description embedded in the original TTM tag metadata.
var ttmCatalog = map[string]ttmCatalogItem{
	"FIRE.TTM":     {"Campfire Effects", "Small through extra-large flame, smoke, wood, and rubbing sticks"},
	"FISHWALK.TTM": {"Fishing Walk", "Fishwalk"},
	"GFFFOOD.TTM":  {"Food Gag", "Load food"},
	"GJCATCH2.TTM": {"Catch Gag", "Catch gag and shaking fist"},
	"GJDIVE.TTM":   {"Diving Gags", "Belly flop, flip, cannonball, and bubbles"},
	"GJGULIVR.TTM": {"Gulliver and the Lilliputians", "Sleeping, tied up, and Lilliputians sailing in and ashore"},
	"GJGULL1.TTM":  {"The Seagull and the Book", "Seagull, book, cleaning, and Johnny getting mad"},
	"GJHOT.TTM":    {"Hot Summer Day", "Hot summer day, fan speeds, and Johnny crumbling"},
	"GJLILIPU.TTM": {"Lilliputian Attack", "Lilliputians sail in, cannon fire, and planes launch"},
	"GJNAT1.TTM":   {"Johnny's Rain Dance", "Rain cloud, light-bulb idea, frenzied dance, and rain"},
	"GJNAT3.TTM":   {"Native Boat Visit", "Boat arrives, Johnny is undressed, and boat leaves"},
	"GJVIS3.TTM":   {"Submarine and Aircraft Visitors", "Periscope, plane, helicopter, and Johnny shrugging"},
	"GJVIS5.TTM":   {"Johnny Jumps at a Plane", "Jumping Johnny, collision, and plane starting"},
	"GJVIS5W.TTM":  {"Plane Jump — Short Version", "Load visitor 5, jumping Johnny, and collision"},
	"GJVIS6.TTM":   {"Tanker Visit", "Johnny watches and waves as the tanker arrives"},
	"MEANWHIL.TTM": {"Quarky Watch", "Quarky watch"},
	"MJAMBWLK.TTM": {"Ambient Walking", "Ambient walk, foot, look, and standing sequences at island spots"},
	"MJBATH.TTM":   {"Ocean Bath and Stolen Clothes", "Bathing, hair washing, and a gull taking Johnny's clothes"},
	"MJCOCO.TTM":   {"Chasing the Coconut", "Shake tree, falling coconut, bounce, chase, and smash"},
	"MJCOCO1.TTM":  {"Eating the Coconut", "Chase, break, eat, chew, and big sigh"},
	"MJDIVE.TTM":   {"Johnny Goes Diving", "Dive, bubbles, and walking out of the water"},
	"MJFIRE.TTM":   {"Building a Campfire", "Rubbing sticks, growing fire, cooking, eating, and dying embers"},
	"MJFISH.TTM":   {"Fishing by the Tree", "Cast, reel, catches, crab, boot, octopus, and tree sequences"},
	"MJFISHC.TTM":  {"Fishing from the Shore", "Casting, reeling, catches, starfish, crab, boot, and large fish"},
	"MJJOG.TTM":    {"Johnny Goes Jogging", "Stretching, running, out of breath, and the last leg"},
	"MJRAFT.TTM":   {"Building the Raft", "Getting boards, building, standing, and dusting off hands"},
	"MJREAD.TTM":   {"Reading with the Seagull", "Reading, page turns, sleep, coconut bump, and gull stealing the book"},
	"MJSAND.TTM":   {"Building a Sandcastle", "Castle construction, kicking, Lilliputians, planes, and King Kong routine"},
	"MJTELE.TTM":   {"Looking Through the Telescope", "Lift, scan left/right, shifting eye, and lower telescope"},
	"SASKDATE.TTM": {"Johnny Asks Mary on a Date", "Mary approaches, Johnny asks, Mary accepts, and they wave goodbye"},
	"SBREAKUP.TTM": {"Showing Mary the Raft", "Johnny builds, Mary arrives, Johnny shows the raft, and breakup begins"},
	"SHARK1.TTM":   {"Here Comes the Shark", "Water check and shark arrival"},
	"SJGLIMPS.TTM": {"Johnny's First Glimpse of Mary", "Johnny fishing while Mary swims and dives"},
	"SJLEAVES.TTM": {"Johnny Leaves the Island", "Raft ready, Johnny gets his bags, and leaves"},
	"SJMSSGE.TTM":  {"Message in a Bottle", "Johnny writes a letter, bottles it, throws it, and dreams"},
	"SJMSUZY.TTM":  {"Johnny Meets Suzy", "Suzy meets Johnny"},
	"SJWORK.TTM":   {"Johnny at Work", "Johnny in the office remembers the island"},
	"SMDATE.TTM":   {"Johnny and Mary's Date", "Dancing, eating, toast, drinks, and waving"},
	"SUZYCITY.TTM": {"Suzy's Message", "Tanning oil, floating bottle, first message, and thoughts of the island"},
	"THEEND.TTM":   {"Back on the Island", "Plane drops Johnny, Johnny dances, returns to the island, and credits"},
	"WOULDBE.TTM":  {"The Would-Be Rescuers", "Boat passes, returns, Johnny swims out, and they leave for good"},
}

func ttmCatalogInfo(resourceName string) ttmCatalogItem {
	if item, ok := ttmCatalog[resourceName]; ok {
		return item
	}
	return ttmCatalogItem{label: resourceName, description: "Original animation resource"}
}
