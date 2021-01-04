### Notes on using mongodb  

 - Below is shown how to talk with mongodb via console, to make quick checks on the dataset.  
   So first you better check the monogodb [docs](https://docs.mongodb.com/manual/mongo/).  
   Then get inside the docker:  
   ```
   docker exec -ti mongo mongo
   ```  
   Select needed db:  
   ```
   show dbs
   use ann_bench
   ```  
   You can create/drop indexes:  
   ```
   db.train.createIndex({Info: 1})
   # index on array may take much time
   db.train.createIndex({featureVec: 1})
   db.train.dropIndex({featureVec: 1})
   ```  
   Empty find query will return all records, bounded by the limit value:  
   ```
   db.train.find().limit(2)
   ```  
   Also extra-useful thing is query analysis:  
   ```
   db.train.find("secondaryId": 1).limit(2).explain("executionStats)
   ```  
   Clean the collection:  
   ```
   db.train.remove({})
   ```  
   Make aggregations, like getting mean and std vectors on the random data sample:  
   ```
   db.train.aggregate([
     {
       $sample: {
         size: 100000
       }
     },
     {
       $unwind: {
         path: "$featureVec",
         includeArrayIndex: "i"
       }
     },
     {
       $group: {
         _id: "$i",
         avg: {
           $avg: "$featureVec"
         },
         std: {
           $stdDevSamp: "$featureVec"
         }
       }
     },
     {
       $sort: {
         "_id": 1
       }
     },
     {
       $group: {
         _id: null,
         avg: {
           $push: "$avg"
         },
         std: {
           $push: "$std"
         }
       }
     }
   ])
   ```  
 - The mongodb go client is a connection pool already so it is thread safe: https://github.com/mongodb/mongo-go-driver/blob/master/mongo/client.go#L42  
 Quote from the code:  
```
 // Client is a handle representing a pool of connections to a MongoDB deployment. It is safe for concurrent use by
 // multiple goroutines.
 //
 // The Client type opens and closes connections automatically and maintains a pool of idle connections. For
 // connection pool configuration options, see documentation for the ClientOptions type in the mongo/options package.
```  
 - use mongo's `find` only with limiting, otherwise - db starts lagging. Not sure why...;  
 - monitor mongodb mem usage:  
 ```
 db.serverStatus().mem
    {
    	"bits" : 64,
    	"resident" : 907,
    	"virtual" : 1897,
    	"supported" : true,
    	"mapped" : 0,
    	"mappedWithJournal" : 0
    }
```  
 - if the mongo consumes too much ram while running inside the docker - just try to specify the WiredTiger mem cache  `-wiredTigerCacheSizeGB 2.5` to some lower value, like `(docker_mem_limit - 1) / 2`;  
 - don't forget to define indexes. In my case its `SecondaryID` and `Hashes.hash#` fields;  