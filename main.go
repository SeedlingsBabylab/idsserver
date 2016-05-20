package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	/*
		labsDB is the wrapper around the lab user info database.
		It keeps track of all the users and the current jobs that
		have been handed out to them.
	*/
	labsDB *LabsDB

	/*
		workDB is the wrapper around the database that keeps track of
		all the jobs that have been assigned and yet to be assigned.
	*/
	workDB *WorkDB

	/*
		labelsDB is the wrapper around the database that stores all the
		classifications that the users send back to the server. Keys are
		Block ID's, values are Block JSONs
	*/
	labelsDB *LabelsDB
	/*
		manifestFile is the path to the path_manifests.csv file.
		This contains the name of the clan files and the paths to
		all the blocks that are a part of them.

		format:

		[clan_file, block_index, path_to_block]

	*/
	manifestFile string

	/*
		configFile is the path to the main config file,
		which has the server admin keys and other metadata
	*/
	configFile string

	/*
		mainConfig is the Config struct produced from reading
		the configFile
	*/
	mainConfig Config

	/*
		dataMap is the global map of CLAN files to block paths
	*/
	dataMap DataMap

	/*
		workItemMap is a map of WorkItem to bool.
		The boolean value represents whether or not
		the particular work item is active or not.
		Active means it's been sent out for coding
		and has not been submitted back yet.
	*/
	workItemMap WorkItemMap

	/*
		activeWorkItems is a map of WorkItem ID's. All the ID's
		represent blocks which have been sent out to be worked on.
		(i.e. active blocks)

		format:

		map["30_13_coderJS_final-2:::6" : true , "32_13_coderJS_final-5:::16" : true, ......]

		The ID's are a concatenation of the name of the CLAN file of origin
		and the block index, separated by ":::".
	*/
	activeWorkItems ActiveDataQueue
)

const (
	/*
		dataPath is the path to where all the
		CLAN files and audio blocks are going
		to be stored
	*/
	dataPath = "data"

	/*
		numBlocksToSend is the number of blocks that will be sent
		from any given CLAN file to the end user upon request
	*/
	numBlocksToSend = 5
)

/*
Config is a struct representing metadata about
the configuration state of the server upon startup.
It's loaded from the config.json file read in as argument
from the command line upon starting the server.
*/
type Config struct {
	AdminKey      string `json:"admin-key"`
	WorkMapLoaded bool   `json:"work-map-loaded"`
}

func (conf *Config) encode() ([]byte, error) {
	enc, err := json.MarshalIndent(conf, "", " ")
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func (conf *Config) writeFile() {
	fmt.Println("tried to update the config file")
	encodedConf, err := conf.encode()
	if err != nil {
		log.Fatal(err)
	}
	writeErr := ioutil.WriteFile(configFile, encodedConf, 0644)
	if err != nil {
		log.Fatal(writeErr)
	}
}

func readConfigFile(path string) Config {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	json.Unmarshal(file, &config)

	return config
}

func shutDown() {

}

func main() {

	configFile = os.Args[1]
	manifestFile = os.Args[2]

	// Open the LabsDB
	labsDB = LoadLabsDB()
	defer labsDB.Close()

	workDB = LoadWorkDB()
	defer workDB.Close()

	labelsDB = LoadLabelsDB()
	defer labelsDB.Close()

	mainConfig = readConfigFile(configFile)

	fmt.Println("mainConfig: ")
	fmt.Println(mainConfig)

	//	return

	dataMap := fillDataMap()

	/*
		get the WorkItemMap, either from the dataMap,
		or from the workDB on disk.
	*/
	if !mainConfig.WorkMapLoaded {
		workItemMap = dataMap.partitionIntoWorkItemsMap()
		workDB.persistWorkItemMap(workItemMap)
		mainConfig.WorkMapLoaded = true
		//mainConfig.writeFile()
	} else {
		workItemMap = workDB.loadItemMap()
	}

	fmt.Println("# of work items map: ", len(workItemMap))

	// for key, value := range workItemMap {
	//
	// 	fmt.Println(key)
	// 	fmt.Println(value)
	// }

	/*
		labsDB.addUser("123456", "Bergelson Lab", "andrei")
		labsDB.addUser("123456", "Bergelson Lab", "shannon")
		labsDB.addUser("123457", "Einstein Lab", "alice")
		labsDB.addUser("123458", "Fermi Lab", "bob")
		labsDB.addUser("123459", "Soderstrom Lab", "sally")
		labsDB.addUser("123450", "Marr Lab", "joe")
		labsDB.addUser("123450", "Marr Lab", "fred")
		labsDB.addUser("123450", "Marr Lab", "jane")
		labsDB.addUser("123450", "Marr Lab", "bill")
		labsDB.addUser("1234567654321", "Billy's lab", "billybob")
	*/
	//labs := labsDB.getAllLabs()

	// fmt.Println("Printing all the labs: ")
	// for _, lab := range labs {
	// 	fmt.Println(*lab)
	// }

	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/getblock/", getBlockHandler)
	http.HandleFunc("/labinfo/", labInfoHandler)
	http.HandleFunc("/alllabinfo/", allLabInfoHandler)
	http.HandleFunc("/adduser/", addUserHandler)
	http.HandleFunc("/submitlabels/", submitLabelsHandler)
	http.HandleFunc("/shutdown/", shutDownHandler)

	http.ListenAndServe(":8080", nil)

}
