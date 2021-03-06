package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"

	"github.com/decibelcooper/proio/go-proio"
	"github.com/decibelcooper/proio/go-proio/model"
	prolcio "github.com/decibelcooper/proio/go-proio/model/lcio"
	"go-hep.org/x/hep/lcio"
)

var (
	outFile = flag.String("o", "", "file to save output to")
	doGzip  = flag.Bool("g", false, "compress the stdout output with gzip")
)

func printUsage() {
	fmt.Fprintf(os.Stderr,
		`Usage: lcio2proio [options] <lcio-input-file>
options:
`,
	)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = printUsage
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		log.Fatal("Invalid arguments")
	}

	lcioReader, err := lcio.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer lcioReader.Close()

	var proioWriter *proio.Writer
	if *outFile == "" {
		if *doGzip {
			proioWriter = proio.NewGzipWriter(os.Stdout)
		} else {
			proioWriter = proio.NewWriter(os.Stdout)
		}
	} else {
		proioWriter, err = proio.Create(*outFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	defer proioWriter.Close()

	for lcioReader.Next() {
		lcioEvent := lcioReader.Event()
		proioEvent := proio.NewEvent()

		proioEvent.Header.RunNumber = uint64(lcioEvent.RunNumber)
		proioEvent.Header.EventNumber = uint64(lcioEvent.EventNumber)

		for i, collName := range lcioEvent.Names() {
			lcioColl := lcioEvent.Get(collName)

			var proioColl proio.Collection
			switch lcioColl.(type) {
			case *lcio.McParticleContainer:
				proioColl = convertMCParticleCollection(lcioColl.(*lcio.McParticleContainer), &lcioEvent, uint32(i+1))
			case *lcio.SimTrackerHitContainer:
				proioColl = convertSimTrackerHitCollection(lcioColl.(*lcio.SimTrackerHitContainer), &lcioEvent, uint32(i+1))
			case *lcio.TrackerRawDataContainer:
				proioColl = convertTrackerRawDataCollection(lcioColl.(*lcio.TrackerRawDataContainer), &lcioEvent, uint32(i+1))
			case *lcio.TrackerDataContainer:
				proioColl = convertTrackerDataCollection(lcioColl.(*lcio.TrackerDataContainer), &lcioEvent, uint32(i+1))
			case *lcio.TrackerHitContainer:
				proioColl = convertTrackerHitCollection(lcioColl.(*lcio.TrackerHitContainer), &lcioEvent, uint32(i+1))
			case *lcio.TrackerPulseContainer:
				proioColl = convertTrackerPulseCollection(lcioColl.(*lcio.TrackerPulseContainer), &lcioEvent, uint32(i+1))
			case *lcio.TrackerHitPlaneContainer:
				proioColl = convertTrackerHitPlaneCollection(lcioColl.(*lcio.TrackerHitPlaneContainer), &lcioEvent, uint32(i+1))
			case *lcio.TrackerHitZCylinderContainer:
				proioColl = convertTrackerHitZCylinderCollection(lcioColl.(*lcio.TrackerHitZCylinderContainer), &lcioEvent, uint32(i+1))
			case *lcio.TrackContainer:
				proioColl = convertTrackCollection(lcioColl.(*lcio.TrackContainer), &lcioEvent, uint32(i+1))
			case *lcio.SimCalorimeterHitContainer:
				proioColl = convertSimCalorimeterHitCollection(lcioColl.(*lcio.SimCalorimeterHitContainer), &lcioEvent, uint32(i+1))
			case *lcio.RawCalorimeterHitContainer:
				proioColl = convertRawCalorimeterHitCollection(lcioColl.(*lcio.RawCalorimeterHitContainer), &lcioEvent, uint32(i+1))
			case *lcio.CalorimeterHitContainer:
				proioColl = convertCalorimeterHitCollection(lcioColl.(*lcio.CalorimeterHitContainer), &lcioEvent, uint32(i+1))
			case *lcio.ClusterContainer:
				proioColl = convertClusterCollection(lcioColl.(*lcio.ClusterContainer), &lcioEvent, uint32(i+1))
			case *lcio.RecParticleContainer:
				proioColl = convertRecParticleCollection(lcioColl.(*lcio.RecParticleContainer), &lcioEvent, uint32(i+1))
			case *lcio.VertexContainer:
				proioColl = convertVertexCollection(lcioColl.(*lcio.VertexContainer), &lcioEvent, uint32(i+1))
			case *lcio.RelationContainer:
				proioColl = convertRelationCollection(lcioColl.(*lcio.RelationContainer), &lcioEvent, uint32(i+1))
			}

			if proioColl != nil {
				if err := proioEvent.Add(proioColl, collName); err != nil {
					log.Fatal("Failed to add collection ", collName, ": ", err)
				}
			}
		}

		proioWriter.Push(proioEvent)
	}

	err = lcioReader.Err()
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
}

func convertIntParams(intParams map[string][]int32) map[string]*model.IntParams {
	params := map[string]*model.IntParams{}
	for key, value := range intParams {
		params[key] = &model.IntParams{Array: value}
	}
	return params
}

func convertFloatParams(floatParams map[string][]float32) map[string]*model.FloatParams {
	params := map[string]*model.FloatParams{}
	for key, value := range floatParams {
		params[key] = &model.FloatParams{Array: value}
	}
	return params
}

func convertStringParams(stringParams map[string][]string) map[string]*model.StringParams {
	params := map[string]*model.StringParams{}
	for key, value := range stringParams {
		params[key] = &model.StringParams{Array: value}
	}
	return params
}

func convertParams(lcioParams lcio.Params) *model.Params {
	return &model.Params{
		Ints:    convertIntParams(lcioParams.Ints),
		Floats:  convertFloatParams(lcioParams.Floats),
		Strings: convertStringParams(lcioParams.Strings),
	}
}

func makeRef(entry interface{}, event *lcio.Event) *model.Reference {
	for i, collName := range event.Names() {
		collGen := event.Get(collName)

		j := 0
		found := false
		switch collGen.(type) {
		case *lcio.McParticleContainer:
			coll := collGen.(*lcio.McParticleContainer)
			for j = range coll.Particles {
				if &coll.Particles[j] == entry {
					found = true
					break
				}
			}
		case *lcio.TrackerRawDataContainer:
			coll := collGen.(*lcio.TrackerRawDataContainer)
			for j = range coll.Data {
				if &coll.Data[j] == entry {
					found = true
					break
				}
			}
		case *lcio.TrackerDataContainer:
			coll := collGen.(*lcio.TrackerDataContainer)
			for j = range coll.Data {
				if &coll.Data[j] == entry {
					found = true
					break
				}
			}
		case *lcio.RawCalorimeterHitContainer:
			coll := collGen.(*lcio.RawCalorimeterHitContainer)
			for j = range coll.Hits {
				if &coll.Hits[j] == entry {
					found = true
					break
				}
			}
		case *lcio.TrackContainer:
			coll := collGen.(*lcio.TrackContainer)
			for j = range coll.Tracks {
				if &coll.Tracks[j] == entry {
					found = true
					break
				}
			}
		case *lcio.TrackerHitContainer:
			coll := collGen.(*lcio.TrackerHitContainer)
			for j = range coll.Hits {
				if &coll.Hits[j] == entry {
					found = true
					break
				}
			}
		case *lcio.ClusterContainer:
			coll := collGen.(*lcio.ClusterContainer)
			for j = range coll.Clusters {
				if &coll.Clusters[j] == entry {
					found = true
					break
				}
			}
		case *lcio.CalorimeterHitContainer:
			coll := collGen.(*lcio.CalorimeterHitContainer)
			for j = range coll.Hits {
				if &coll.Hits[j] == entry {
					found = true
					break
				}
			}
		case *lcio.RecParticleContainer:
			coll := collGen.(*lcio.RecParticleContainer)
			for j = range coll.Parts {
				if &coll.Parts[j] == entry {
					found = true
					break
				}
			}
		case *lcio.VertexContainer:
			coll := collGen.(*lcio.VertexContainer)
			for j = range coll.Vtxs {
				if &coll.Vtxs[j] == entry {
					found = true
					break
				}
			}
		}

		if found {
			return &model.Reference{
				CollID:  uint32(i + 1),
				EntryID: uint32(j + 1),
			}
		}
	}
	return nil
}

func makeRefs(entries interface{}, event *lcio.Event) []*model.Reference {
	slice := reflect.ValueOf(entries)
	refs := make([]*model.Reference, 0)
	for i := 0; i < slice.Len(); i++ {
		ref := makeRef(slice.Index(i).Interface(), event)
		if ref != nil {
			refs = append(refs, ref)
		}
	}
	return refs
}

func convertMCParticleCollection(lcioColl *lcio.McParticleContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.MCParticleCollection {
	proioColl := &prolcio.MCParticleCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Particles {
		proioEntry := &prolcio.MCParticle{
			Id:        uint32(i + 1),
			Parents:   makeRefs(lcioEntry.Parents, lcioEvent),
			Children:  makeRefs(lcioEntry.Children, lcioEvent),
			PDG:       lcioEntry.PDG,
			GenStatus: lcioEntry.GenStatus,
			SimStatus: lcioEntry.SimStatus,
			Vertex:    lcioColl.Particles[i].Vertex[:],
			Time:      lcioEntry.Time,
			P:         lcioColl.Particles[i].P[:],
			Mass:      lcioEntry.Mass,
			Charge:    lcioEntry.Charge,
			PEndPoint: lcioColl.Particles[i].PEndPoint[:],
			Spin:      lcioColl.Particles[i].Spin[:],
			ColorFlow: lcioColl.Particles[i].ColorFlow[:],
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertSimTrackerHitCollection(lcioColl *lcio.SimTrackerHitContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.SimTrackerHitCollection {
	proioColl := &prolcio.SimTrackerHitCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Hits {
		proioEntry := &prolcio.SimTrackerHit{
			Id:         uint32(i + 1),
			CellID0:    lcioEntry.CellID0,
			CellID1:    lcioEntry.CellID1,
			Pos:        lcioColl.Hits[i].Pos[:],
			EDep:       lcioEntry.EDep,
			Time:       lcioEntry.Time,
			Mc:         makeRef(lcioEntry.Mc, lcioEvent),
			P:          lcioColl.Hits[i].Momentum[:],
			PathLength: lcioEntry.PathLength,
			Quality:    lcioEntry.Quality,
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func copyUint16SliceToUint32(origSlice []uint16) []uint32 {
	slice := make([]uint32, 0)
	for _, value := range origSlice {
		slice = append(slice, uint32(value))
	}
	return slice
}

func convertTrackerRawDataCollection(lcioColl *lcio.TrackerRawDataContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.TrackerRawDataCollection {
	proioColl := &prolcio.TrackerRawDataCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Data {
		proioEntry := &prolcio.TrackerRawData{
			Id:      uint32(i + 1),
			CellID0: lcioEntry.CellID0,
			CellID1: lcioEntry.CellID1,
			Time:    lcioEntry.Time,
			ADCs:    copyUint16SliceToUint32(lcioEntry.ADCs),
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertTrackerDataCollection(lcioColl *lcio.TrackerDataContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.TrackerDataCollection {
	proioColl := &prolcio.TrackerDataCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Data {
		proioEntry := &prolcio.TrackerData{
			Id:      uint32(i + 1),
			CellID0: lcioEntry.CellID0,
			CellID1: lcioEntry.CellID1,
			Time:    lcioEntry.Time,
			Charges: lcioEntry.Charges,
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertTrackerHitCollection(lcioColl *lcio.TrackerHitContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.TrackerHitCollection {
	proioColl := &prolcio.TrackerHitCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Hits {
		proioEntry := &prolcio.TrackerHit{
			Id:      uint32(i + 1),
			CellID0: lcioEntry.CellID0,
			CellID1: lcioEntry.CellID1,
			Pos:     lcioColl.Hits[i].Pos[:],
			Cov:     lcioColl.Hits[i].Cov[:],
			Type:    lcioEntry.Type,
			EDep:    lcioEntry.EDep,
			EDepErr: lcioEntry.EDepErr,
			Time:    lcioEntry.Time,
			Quality: lcioEntry.Quality,
			RawHits: makeRefs(lcioEntry.RawHits, lcioEvent),
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertTrackerPulseCollection(lcioColl *lcio.TrackerPulseContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.TrackerPulseCollection {
	proioColl := &prolcio.TrackerPulseCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Pulses {
		proioEntry := &prolcio.TrackerPulse{
			Id:      uint32(i + 1),
			CellID0: lcioEntry.CellID0,
			CellID1: lcioEntry.CellID1,
			Time:    lcioEntry.Time,
			Charge:  lcioEntry.Charge,
			Cov:     lcioColl.Pulses[i].Cov[:],
			Quality: lcioEntry.Quality,
			TPC:     makeRef(lcioEntry.TPC, lcioEvent),
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertTrackerHitPlaneCollection(lcioColl *lcio.TrackerHitPlaneContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.TrackerHitPlaneCollection {
	proioColl := &prolcio.TrackerHitPlaneCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Hits {
		proioEntry := &prolcio.TrackerHitPlane{
			Id:      uint32(i + 1),
			CellID0: lcioEntry.CellID0,
			CellID1: lcioEntry.CellID1,
			Type:    lcioEntry.Type,
			Pos:     lcioColl.Hits[i].Pos[:],
			U:       lcioColl.Hits[i].U[:],
			V:       lcioColl.Hits[i].V[:],
			DU:      lcioEntry.DU,
			DV:      lcioEntry.DV,
			EDep:    lcioEntry.EDep,
			EDepErr: lcioEntry.EDepErr,
			Time:    lcioEntry.Time,
			Quality: lcioEntry.Quality,
			RawHits: makeRefs(lcioEntry.RawHits, lcioEvent),
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertTrackerHitZCylinderCollection(lcioColl *lcio.TrackerHitZCylinderContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.TrackerHitZCylinderCollection {
	proioColl := &prolcio.TrackerHitZCylinderCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Hits {
		proioEntry := &prolcio.TrackerHitZCylinder{
			Id:      uint32(i + 1),
			CellID0: lcioEntry.CellID0,
			CellID1: lcioEntry.CellID1,
			Type:    lcioEntry.Type,
			Pos:     lcioColl.Hits[i].Pos[:],
			Center:  lcioColl.Hits[i].Center[:],
			DRPhi:   lcioEntry.DRPhi,
			DZ:      lcioEntry.DZ,
			EDep:    lcioEntry.EDep,
			EDepErr: lcioEntry.EDepErr,
			Time:    lcioEntry.Time,
			Quality: lcioEntry.Quality,
			RawHits: makeRefs(lcioEntry.RawHits, lcioEvent),
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertTrackStates(lcioStates []lcio.TrackState) []*prolcio.Track_TrackState {
	slice := make([]*prolcio.Track_TrackState, 0)
	for _, state := range lcioStates {
		slice = append(slice, &prolcio.Track_TrackState{
			Loc:   state.Loc,
			D0:    state.D0,
			Phi:   state.Phi,
			Omega: state.Omega,
			Z0:    state.Z0,
			TanL:  state.TanL,
			Cov:   state.Cov[:],
			Ref:   state.Ref[:],
		})
	}
	return slice
}

func convertTrackCollection(lcioColl *lcio.TrackContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.TrackCollection {
	proioColl := &prolcio.TrackCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Tracks {
		proioEntry := &prolcio.Track{
			Id:         uint32(i + 1),
			Type:       lcioEntry.Type,
			Chi2:       lcioEntry.Chi2,
			NDF:        lcioEntry.NdF,
			DEdx:       lcioEntry.DEdx,
			DEdxErr:    lcioEntry.DEdxErr,
			Radius:     lcioEntry.Radius,
			SubDetHits: lcioEntry.SubDetHits,
			States:     convertTrackStates(lcioEntry.States),
			Tracks:     makeRefs(lcioEntry.Tracks, lcioEvent),
			Hits:       makeRefs(lcioEntry.Hits, lcioEvent),
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertContribs(lcioContribs []lcio.Contrib, lcioEvent *lcio.Event) []*prolcio.SimCalorimeterHit_Contrib {
	slice := make([]*prolcio.SimCalorimeterHit_Contrib, 0)
	for _, contrib := range lcioContribs {
		slice = append(slice, &prolcio.SimCalorimeterHit_Contrib{
			MCParticle: makeRef(contrib.Mc, lcioEvent),
			Energy:     contrib.Energy,
			Time:       contrib.Time,
			PDG:        contrib.PDG,
			StepPos:    contrib.StepPos[:],
		})
	}
	return slice
}

func convertSimCalorimeterHitCollection(lcioColl *lcio.SimCalorimeterHitContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.SimCalorimeterHitCollection {
	proioColl := &prolcio.SimCalorimeterHitCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Hits {
		proioEntry := &prolcio.SimCalorimeterHit{
			Id:            uint32(i + 1),
			CellID0:       lcioEntry.CellID0,
			CellID1:       lcioEntry.CellID1,
			Energy:        lcioEntry.Energy,
			Pos:           lcioColl.Hits[i].Pos[:],
			Contributions: convertContribs(lcioEntry.Contributions, lcioEvent),
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertRawCalorimeterHitCollection(lcioColl *lcio.RawCalorimeterHitContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.RawCalorimeterHitCollection {
	proioColl := &prolcio.RawCalorimeterHitCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Hits {
		proioEntry := &prolcio.RawCalorimeterHit{
			Id:        uint32(i + 1),
			CellID0:   lcioEntry.CellID0,
			CellID1:   lcioEntry.CellID1,
			Amplitude: lcioEntry.Amplitude,
			TimeStamp: lcioEntry.TimeStamp,
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertCalorimeterHitCollection(lcioColl *lcio.CalorimeterHitContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.CalorimeterHitCollection {
	proioColl := &prolcio.CalorimeterHitCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Hits {
		lcioRawHit := lcioEntry.Raw
		var rawHit *model.Reference
		if lcioRawHit != nil {
			rawHit = makeRef(lcioEntry.Raw.(*lcio.RawCalorimeterHit), lcioEvent)
		}

		proioEntry := &prolcio.CalorimeterHit{
			Id:        uint32(i + 1),
			CellID0:   lcioEntry.CellID0,
			CellID1:   lcioEntry.CellID1,
			Energy:    lcioEntry.Energy,
			EnergyErr: lcioEntry.EnergyErr,
			Time:      lcioEntry.Time,
			Pos:       lcioColl.Hits[i].Pos[:],
			Type:      lcioEntry.Type,
			Raw:       rawHit,
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertParticleID(pid *lcio.ParticleID) *prolcio.ParticleID {
	return &prolcio.ParticleID{
		Likelihood: pid.Likelihood,
		Type:       pid.Type,
		PDG:        pid.PDG,
		AlgType:    pid.AlgType,
		Params:     pid.Params,
	}
}

func convertParticleIDs(lcioParticleIDs []lcio.ParticleID) []*prolcio.ParticleID {
	slice := make([]*prolcio.ParticleID, 0)
	for _, pid := range lcioParticleIDs {
		slice = append(slice, convertParticleID(&pid))
	}
	return slice
}

func convertClusterCollection(lcioColl *lcio.ClusterContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.ClusterCollection {
	proioColl := &prolcio.ClusterCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Clusters {
		proioEntry := &prolcio.Cluster{
			Id:         uint32(i + 1),
			Type:       lcioEntry.Type,
			Energy:     lcioEntry.Energy,
			EnergyErr:  lcioEntry.EnergyErr,
			Pos:        lcioColl.Clusters[i].Pos[:],
			PosErr:     lcioColl.Clusters[i].PosErr[:],
			Theta:      lcioEntry.Theta,
			Phi:        lcioEntry.Phi,
			DirErr:     lcioColl.Clusters[i].DirErr[:],
			Shape:      lcioColl.Clusters[i].Shape[:],
			PIDs:       convertParticleIDs(lcioEntry.PIDs),
			Clusters:   makeRefs(lcioEntry.Clusters, lcioEvent),
			Hits:       makeRefs(lcioEntry.Clusters, lcioEvent),
			Weights:    lcioColl.Clusters[i].Weights[:],
			SubDetEnes: lcioColl.Clusters[i].SubDetEnes[:],
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func findParticleID(pids []lcio.ParticleID, pid *lcio.ParticleID) int32 {
	for i := range pids {
		if &pids[i] == pid {
			return int32(i)
		}
	}
	return -1
}

func convertRecParticleCollection(lcioColl *lcio.RecParticleContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.RecParticleCollection {
	proioColl := &prolcio.RecParticleCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Parts {
		proioEntry := &prolcio.RecParticle{
			Id:            uint32(i + 1),
			Type:          lcioEntry.Type,
			P:             lcioColl.Parts[i].P[:],
			Energy:        lcioEntry.Energy,
			Cov:           lcioColl.Parts[i].Cov[:],
			Mass:          lcioEntry.Mass,
			Charge:        lcioEntry.Charge,
			Ref:           lcioColl.Parts[i].Ref[:],
			PIDs:          convertParticleIDs(lcioEntry.PIDs),
			PIDUsed:       findParticleID(lcioEntry.PIDs, lcioEntry.PIDUsed),
			GoodnessOfPID: lcioEntry.GoodnessOfPID,
			Recs:          makeRefs(lcioEntry.Recs, lcioEvent),
			Tracks:        makeRefs(lcioEntry.Tracks, lcioEvent),
			Clusters:      makeRefs(lcioEntry.Clusters, lcioEvent),
			StartVtx:      makeRef(lcioEntry.StartVtx, lcioEvent),
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertVertexCollection(lcioColl *lcio.VertexContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.VertexCollection {
	proioColl := &prolcio.VertexCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Vtxs {
		proioEntry := &prolcio.Vertex{
			Id:      uint32(i + 1),
			Primary: lcioEntry.Primary,
			AlgType: lcioEntry.AlgType,
			Chi2:    lcioEntry.Chi2,
			Prob:    lcioEntry.Prob,
			Pos:     lcioColl.Vtxs[i].Pos[:],
			Cov:     lcioColl.Vtxs[i].Cov[:],
			Params:  lcioEntry.Params,
			RecPart: makeRef(lcioEntry.RecPart, lcioEvent),
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}

func convertRelationCollection(lcioColl *lcio.RelationContainer, lcioEvent *lcio.Event, collID uint32) *prolcio.RelationCollection {
	proioColl := &prolcio.RelationCollection{
		Id:     collID,
		Flags:  uint32(lcioColl.Flags),
		Params: convertParams(lcioColl.Params),
	}

	for i, lcioEntry := range lcioColl.Rels {
		proioEntry := &prolcio.Relation{
			Id:     uint32(i + 1),
			From:   makeRef(lcioEntry.From, lcioEvent),
			To:     makeRef(lcioEntry.To, lcioEvent),
			Weight: lcioEntry.Weight,
		}

		proioColl.Entries = append(proioColl.Entries, proioEntry)
	}

	return proioColl
}
