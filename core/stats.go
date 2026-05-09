package core

// KeyspaceStat holds per-database metrics that the INFO command
// reports under the "# Keyspace" section. We support 4 logical
// databases (db0..db3); index 0 is the default DB used by all
// gets/sets unless a SELECT is issued.
var KeyspaceStat [4]map[string]int

// UpdateDBStat records the latest value for a metric on a given
// database. The map for the database is lazily allocated on first use.
func UpdateDBStat(num int, metric string, value int) {
	if KeyspaceStat[num] == nil {
		KeyspaceStat[num] = make(map[string]int)
	}
	KeyspaceStat[num][metric] = value
}
