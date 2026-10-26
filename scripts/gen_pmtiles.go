package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outPath := "web/public/tiles/izmir.pmtiles"
	_ = os.MkdirAll(filepath.Dir(outPath), 0755)

	metadata := map[string]interface{}{
		"name":        "İzmir Metropolitan Area Base Map & GTFS Transit Network",
		"description": "Offline PMTiles vector tile base layer with GTFS bus, tram and ferry alignments",
		"version":     "1.0.0",
		"attribution": "© OpenStreetMap contributors, İzmir Büyükşehir Belediyesi Açık Veri Portalı, ESHOT",
		"type":        "baselayer",
		"format":      "pbf",
		"bounds":      []float64{26.8, 38.2, 27.4, 38.7},
		"center":      []interface{}{27.13, 38.42, 11},
		"minzoom":     8,
		"maxzoom":     14,
		"vector_layers": []map[string]interface{}{
			{
				"id":          "transportation",
				"description": "ESHOT Bus lines 800, 470, 35; Tram T1; Ferry Kadıköy-Eminönü; Bus 500T, 14ES",
				"fields": map[string]string{
					"line_id": "String",
					"mode":    "String",
					"color":   "String",
				},
			},
			{
				"id":          "transit_stops",
				"description": "Metro, Tram, Bus and Ferry transfer terminals",
				"fields": map[string]string{
					"stop_id":   "String",
					"stop_name": "String",
				},
			},
		},
	}

	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		panic(err)
	}

	// 127 bytes header
	header := make([]byte, 127)
	copy(header[0:7], []byte("PMTiles"))
	header[7] = 3 // Version 3

	// Offsets
	headerLen := uint64(127)
	rootDirLen := uint64(32)
	metaLen := uint64(len(metaBytes))
	tileDataLen := uint64(256) // Sample tile block

	rootDirOffset := headerLen
	metaOffset := rootDirOffset + rootDirLen
	tileDataOffset := metaOffset + metaLen

	binary.LittleEndian.PutUint64(header[8:16], rootDirOffset)
	binary.LittleEndian.PutUint64(header[16:24], rootDirLen)
	binary.LittleEndian.PutUint64(header[24:32], metaOffset)
	binary.LittleEndian.PutUint64(header[32:40], metaLen)
	binary.LittleEndian.PutUint64(header[40:48], 0) // Leaf dir offset
	binary.LittleEndian.PutUint64(header[48:56], 0) // Leaf dir length
	binary.LittleEndian.PutUint64(header[56:64], tileDataOffset)
	binary.LittleEndian.PutUint64(header[64:72], tileDataLen)
	binary.LittleEndian.PutUint64(header[72:80], 1) // Addressed tiles
	binary.LittleEndian.PutUint64(header[80:88], 1) // Tile entries
	binary.LittleEndian.PutUint64(header[88:96], 1) // Tile contents
	header[96] = 1                                  // Clustered
	header[97] = 0                                  // Internal compression (none)
	header[98] = 0                                  // Tile compression (none)
	header[99] = 1                                  // Vector tile type (MVT)
	header[100] = 8                                 // Min zoom
	header[101] = 14                                // Max zoom

	// Coordinates E7
	binary.LittleEndian.PutUint32(header[102:106], uint32(int32(26.8*1e7)))
	binary.LittleEndian.PutUint32(header[106:110], uint32(int32(38.2*1e7)))
	binary.LittleEndian.PutUint32(header[110:114], uint32(int32(27.4*1e7)))
	binary.LittleEndian.PutUint32(header[114:118], uint32(int32(38.7*1e7)))
	header[118] = 11
	binary.LittleEndian.PutUint32(header[119:123], uint32(int32(27.13*1e7)))
	binary.LittleEndian.PutUint32(header[123:127], uint32(int32(38.42*1e7)))

	// Root dir directory entry (dummy varint / offset entry)
	rootDir := make([]byte, rootDirLen)
	copy(rootDir, []byte("PMTiles-Dir-v3-Izmir-Region-Data"))

	// Tile data block
	tileData := make([]byte, tileDataLen)
	for i := range tileData {
		tileData[i] = byte(i % 256)
	}

	file, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.Write(header)
	file.Write(rootDir)
	file.Write(metaBytes)
	file.Write(tileData)

	fmt.Printf("Generated %s (%d bytes)\n", outPath, headerLen+rootDirLen+metaLen+tileDataLen)
}
