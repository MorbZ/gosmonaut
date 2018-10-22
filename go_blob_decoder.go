package gosmonaut

import (
	"./OSMPBF"
	"github.com/golang/protobuf/proto"
)

// goBlobDecoder uses the official Golang Protobuf package.
// All protobuf messages will be unmarshalled to temporary objects before
// processing.
type goBlobDecoder struct {
	q []entityParser
}

func (dec *goBlobDecoder) decode(blob *OSMPBF.Blob, t OSMType) ([]entityParser, OSMTypeSet, error) {
	data, err := getBlobData(blob)
	if err != nil {
		return nil, 0, err
	}

	primitiveBlock := new(OSMPBF.PrimitiveBlock)
	if err := proto.Unmarshal(data, primitiveBlock); err != nil {
		return nil, 0, err
	}

	// Get types
	types := dec.getTypes(primitiveBlock)
	if !types.Get(t) {
		return nil, types, nil
	}

	// Build entity parsers
	dec.q = make([]entityParser, 0, len(primitiveBlock.GetPrimitivegroup()))
	dec.parsePrimitiveBlock(primitiveBlock, t)
	return dec.q, types, nil
}

func (dec *goBlobDecoder) getTypes(pb *OSMPBF.PrimitiveBlock) (types OSMTypeSet) {
	for _, pg := range pb.GetPrimitivegroup() {
		if len(pg.GetNodes()) > 0 || len(pg.GetDense().GetId()) > 0 {
			types.Set(NodeType, true)
		}
		if len(pg.GetWays()) > 0 {
			types.Set(WayType, true)
		}
		if len(pg.GetRelations()) > 0 {
			types.Set(RelationType, true)
		}
	}
	return
}

func (dec *goBlobDecoder) parsePrimitiveBlock(pb *OSMPBF.PrimitiveBlock, t OSMType) {
	for _, pg := range pb.GetPrimitivegroup() {
		switch t {
		case NodeType:
			if len(pg.GetNodes()) > 0 {
				dec.q = append(dec.q, newGoNodeParser(pb, pg.GetNodes()))
			} else if len(pg.GetDense().GetId()) > 0 {
				dec.q = append(dec.q, newGoDenseNodesParser(pb, pg.GetDense()))
			}
		case WayType:
			dec.q = append(dec.q, newGoWayParser(pb, pg.GetWays()))
		case RelationType:
			dec.q = append(dec.q, newGoRelationParser(pb, pg.GetRelations()))
		}
	}
}

/* Node Parsers */
type goNodeParser struct {
	index                int
	granularity          int64
	latOffset, lonOffset int64
	st                   []string
	nodes                []*OSMPBF.Node
}

func newGoNodeParser(pb *OSMPBF.PrimitiveBlock, nodes []*OSMPBF.Node) *goNodeParser {
	return &goNodeParser{
		granularity: int64(pb.GetGranularity()),
		latOffset:   pb.GetLatOffset(),
		lonOffset:   pb.GetLonOffset(),
		st:          pb.GetStringtable().GetS(),
		nodes:       nodes,
	}
}

func (d *goNodeParser) isEntityParser() {}

func (d *goNodeParser) next() (id int64, lat, lon float64, tags OSMTags, ok bool) {
	if d.index >= len(d.nodes) {
		return
	}
	ok = true

	node := d.nodes[d.index]
	id = node.GetId()
	lat = decodeCoord(d.latOffset, d.granularity, node.GetLat())
	lon = decodeCoord(d.latOffset, d.granularity, node.GetLon())
	tags = extractTags(d.st, node.GetKeys(), node.GetVals())

	d.index++
	return
}

type goDenseNodesParser struct {
	index                int
	id, lat, lon         int64
	ids                  []int64
	granularity          int64
	latOffset, lonOffset int64
	lats, lons           []int64
	tu                   *tagUnpacker
}

func newGoDenseNodesParser(pb *OSMPBF.PrimitiveBlock, dn *OSMPBF.DenseNodes) *goDenseNodesParser {
	st := pb.GetStringtable().GetS()
	return &goDenseNodesParser{
		granularity: int64(pb.GetGranularity()),
		latOffset:   pb.GetLatOffset(),
		lonOffset:   pb.GetLonOffset(),
		ids:         dn.GetId(),
		lats:        dn.GetLat(),
		lons:        dn.GetLon(),
		tu:          &tagUnpacker{st, dn.GetKeysVals(), 0},
	}
}

func (d *goDenseNodesParser) isEntityParser() {}

func (d *goDenseNodesParser) next() (id int64, lat, lon float64, tags OSMTags, ok bool) {
	if d.index >= len(d.ids) {
		return
	}
	ok = true

	d.id = d.ids[d.index] + d.id
	id = d.id

	d.lat = d.lats[d.index] + d.lat
	d.lon = d.lons[d.index] + d.lon
	lat = decodeCoord(d.latOffset, d.granularity, d.lat)
	lon = decodeCoord(d.lonOffset, d.granularity, d.lon)
	tags = d.tu.next()

	d.index++
	return
}

/* Way Parser */
type goWayParser struct {
	index int
	st    []string
	ways  []*OSMPBF.Way
}

func newGoWayParser(pb *OSMPBF.PrimitiveBlock, ways []*OSMPBF.Way) *goWayParser {
	return &goWayParser{
		st:   pb.GetStringtable().GetS(),
		ways: ways,
	}
}

func (d *goWayParser) isEntityParser() {}

func (d *goWayParser) next() (id int64, tags OSMTags, ok bool) {
	if d.index >= len(d.ways) {
		return
	}
	ok = true

	way := d.ways[d.index]
	id = way.GetId()
	tags = extractTags(d.st, way.GetKeys(), way.GetVals())

	d.index++
	return
}

func (d *goWayParser) refs() []int64 {
	protoIDs := d.ways[d.index-1].GetRefs()
	ids := make([]int64, len(protoIDs))
	var id int64
	for i, protoID := range protoIDs {
		id += protoID // delta encoding
		ids[i] = id
	}
	return ids
}

/* Relation Parser */
type goRelationParser struct {
	index     int
	st        []string
	relations []*OSMPBF.Relation
}

func newGoRelationParser(pb *OSMPBF.PrimitiveBlock, relations []*OSMPBF.Relation) *goRelationParser {
	return &goRelationParser{
		st:        pb.GetStringtable().GetS(),
		relations: relations,
	}
}

func (d *goRelationParser) isEntityParser() {}

func (d *goRelationParser) next() (id int64, tags OSMTags, ok bool) {
	if d.index >= len(d.relations) {
		return
	}
	ok = true

	rel := d.relations[d.index]
	id = rel.GetId()
	tags = extractTags(d.st, rel.GetKeys(), rel.GetVals())

	d.index++
	return
}

func (d *goRelationParser) ids() []int64 {
	protoIDs := d.relations[d.index-1].GetMemids()
	ids := make([]int64, len(protoIDs))
	var id int64
	for i, protoID := range protoIDs {
		id += protoID
		ids[i] = id
	}
	return ids
}

func (d *goRelationParser) roles() []string {
	protoRoles := d.relations[d.index-1].GetRolesSid()
	roles := make([]string, len(protoRoles))
	for i, protoRole := range protoRoles {
		roles[i] = d.st[protoRole]
	}
	return roles
}

func (d *goRelationParser) types() []OSMType {
	protoTypes := d.relations[d.index-1].GetTypes()
	types := make([]OSMType, len(protoTypes))
	for i, protoT := range protoTypes {
		var t OSMType
		switch protoT {
		case OSMPBF.Relation_NODE:
			t = NodeType
		case OSMPBF.Relation_WAY:
			t = WayType
		case OSMPBF.Relation_RELATION:
			t = RelationType
		}
		types[i] = t
	}
	return types
}

/* Tag Decoding */
// Make tags map from stringtable and two parallel arrays of IDs.
func extractTags(stringTable []string, keyIDs, valueIDs []uint32) OSMTags {
	tags := NewOSMTags(len(keyIDs))
	for index, keyID := range keyIDs {
		key := stringTable[keyID]
		val := stringTable[valueIDs[index]]
		tags.Set(key, val)
	}
	return tags
}

type tagUnpacker struct {
	stringTable []string
	keysVals    []int32
	index       int
}

// Make tags map from stringtable and array of IDs (used in DenseNodes encoding).
func (tu *tagUnpacker) next() OSMTags {
	var tags OSMTags
	for tu.index < len(tu.keysVals) {
		keyID := tu.keysVals[tu.index]
		tu.index++
		if keyID == 0 {
			break
		}

		valID := tu.keysVals[tu.index]
		tu.index++

		key := tu.stringTable[keyID]
		val := tu.stringTable[valID]
		tags.Set(key, val)
	}
	return tags
}