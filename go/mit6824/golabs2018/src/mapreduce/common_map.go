package mapreduce

import (
	"errors"
	"os"
	"hash/fnv"
	"io/ioutil"
	"encoding/json"
)

func doMap(
	jobName string, // the name of the MapReduce job
	mapTask int, // which map task this is
	inFile string,
	nReduce int, // the number of reduce task that will be run ("R" in the paper)
	mapF func(filename string, contents string) []KeyValue,
) {
	mylog("---------doMap begin---------")
	mylog("jobName:", jobName)
	mylog("mapTask:", mapTask)
	mylog("inFile:", inFile)
	mylog("nReduce:", nReduce)
	
	//
	// doMap manages one map task: it should read one of the input files
	// (inFile), call the user-defined map function (mapF) for that file's
	// contents, and partition mapF's output into nReduce intermediate files.
	//
	// There is one intermediate file per reduce task. The file name
	// includes both the map task number and the reduce task number. Use
	// the filename generated by reduceName(jobName, mapTask, r)
	// as the intermediate file for reduce task r. Call ihash() (see
	// below) on each key, mod nReduce, to pick r for a key/value pair.
	//
	// mapF() is the map function provided by the application. The first
	// argument should be the input file name, though the map function
	// typically ignores it. The second argument should be the entire
	// input file contents. mapF() returns a slice containing the
	// key/value pairs for reduce; see common.go for the definition of
	// KeyValue.
	//
	// Look at Go's ioutil and os packages for functions to read
	// and write files.
	//
	// Coming up with a scheme for how to format the key/value pairs on
	// disk can be tricky, especially when taking into account that both
	// keys and values could contain newlines, quotes, and any other
	// character you can think of.
	//
	// One format often used for serializing data to a byte stream that the
	// other end can correctly reconstruct is JSON. You are not required to
	// use JSON, but as the output of the reduce tasks *must* be JSON,
	// familiarizing yourself with it here may prove useful. You can write
	// out a data structure as a JSON string to a file using the commented
	// code below. The corresponding decoding functions can be found in
	// common_reduce.go.
	//
	//   enc := json.NewEncoder(file)
	//   for _, kv := ... {
	//     err := enc.Encode(&kv)
	//
	// Remember to close the file after you have written all the values!
	//
	// Your code here (Part I).
	//

	// 读取文件
	fileContent, err := readFile(inFile)
	if err != nil {
		mylog("readFile error", err)
		return
	}

	// 调用map函数
	var kvList []KeyValue = mapF(inFile, fileContent)
	if kvList == nil {
		mylog("mapF return nil")
		return
	}

	// 遍历map的输出，输出到nReduce个不同的文件中
	reduceList := make([][]KeyValue, nReduce)
	for _, kv := range kvList {
		reduceTask := ihash(kv.Key) % nReduce
		reduceList[reduceTask] = append(reduceList[reduceTask], kv)
	}
	for i := 0; i < nReduce; i++ {
		rname := reduceName(jobName, mapTask, i)
		err := writeFileByJson(rname, reduceList[i])
		if err != nil {
			mylog("writeFileByJson error", err)
		}
		
	}
	
	mylog("---------doMap end---------")
}

func ihash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32() & 0x7fffffff)
}

func readFile(inFile string) (string,error) {
	bytes, err := ioutil.ReadFile(inFile)
	return string(bytes[:]),err
}

func writeFileByJson(fileName string, kvList []KeyValue) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	if encoder == nil {
		return errors.New("NewEncoder return nil")
	}
	for _, kv := range kvList {
		err := encoder.Encode(kv)
		if err != nil {
			mylog("Encode error ", err)
			return err
		}
	}

	return nil
}
