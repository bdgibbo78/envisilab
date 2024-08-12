package core

import (
	"github.com/golang/geo/s2"
)

//
//
//
type Cell struct {

	// The id of the current occupied cell
	s2cellID s2.CellID

	// A List of ids of the the current cell and surrounding cells
	cellGroup []s2.CellID

	// The configuration which details the cell layout
	config *Config
}

func MakeCell(config *Config) *Cell {
	c := new(Cell)
	c.s2cellID = 0
	c.cellGroup = make([]s2.CellID, 0, 9)
	c.config = config
	return c
}

func (c *Cell) Update(cid s2.CellID) {
	c.s2cellID = cid
	c.updateBoundingCells()
}

// Changed
func (c *Cell) Changed(loc *Location) bool {
	s2CellId := s2.CellIDFromLatLng(s2.LatLngFromDegrees(float64(loc.Lat), float64(loc.Lng))).Parent(c.config.topologyLevel)
	if s2CellId != c.s2cellID {
		c.s2cellID = s2CellId
		c.updateBoundingCells()
		return true
	}
	return false
}

//
//
//
func (c *Cell) updateBoundingCells() {
	region := s2.CapFromCenterHeight(s2.PointFromLatLng(c.s2cellID.LatLng().Normalized()), c.config.height)

	rc := &s2.RegionCoverer{MaxLevel: c.config.topologyLevel, MinLevel: c.config.topologyLevel}
	cells := rc.Covering(region)

	// Clear the slice
	c.cellGroup = c.cellGroup[:0]
	for _, cid := range cells {
		c.cellGroup = append(c.cellGroup, cid)
	}
}

func (c *Cell) GetConfig() *Config {
	return c.config
}

func (c *Cell) GetCellId() s2.CellID {
	return c.s2cellID
}

func (c *Cell) GetCellGroup() []s2.CellID {
	return c.cellGroup
}

func CellCenterLocation(cellId s2.CellID) Location {
	s2Cell := s2.CellFromCellID(cellId)
	center := s2.LatLngFromPoint(s2Cell.Center())
	return MakeLocation(center.Lat.Degrees(), center.Lng.Degrees(), 0.0, 0.0, 0)
}
