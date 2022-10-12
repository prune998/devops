# Usage

```shell
go build
rm results/pubsub.txt results/storage.txt results/bigquery.txt
now=$(date) ; echo "resource,kind,project,name,generated $now" > results/pubsub.txt
for i in $(cat ~/Documents/dev/scripts/all-gcp-projects.txt)
do echo $i 
    gcloud pubsub topics list --format json --project $i | ./dasel-tool -kind topics >> results/pubsub.txt
    gcloud pubsub subscriptions list --format json --project $i | ./dasel-tool -kind subscriptions >> results/pubsub.txt
done

now=$(date) ; echo "resource,kind,project,name,generated $now" > results/storage.txt
for i in $(cat ~/Documents/dev/scripts/all-gcp-projects.txt) ; do echo $i ;gcloud alpha storage ls -j --project $i  --quiet| ./dasel-tool -kind storage --fetchProjects >> results/storage.txt ; done

now=$(date) ; echo "resource,kind,project,name,generated $now" > results/bigquery.txt
for i in $(cat ~/Documents/dev/scripts/all-gcp-projects.txt) ; do echo $i ;gcloud alpha bq datasets list --format=json --project $i  --quiet| ./dasel-tool -kind datasets >> results/bigquery.txt ; done

now=$(date) ; echo "resource,kind,project,region,name,generated $now" > results/redis.txt
for i in $(cat ~/Documents/dev/scripts/all-gcp-projects.txt) ; do for r in us-east4 us-central1 europe-west2 us-west1 ; do echo "$i/$r" ;gcloud redis instances list --region $r --format=json --project $i --quiet | ./dasel-tool -kind redis >> results/redis.txt ; done ; done

now=$(date) ; echo "resource,kind,project,region,name,generated $now" > results/functions.txt
for i in $(cat ~/Documents/dev/scripts/all-gcp-projects.txt) ; do echo "$i" ;gcloud functions list --format=json --project $i --quiet | ./dasel-tool -kind functions >> results/functions.txt ; done 

now=$(date) ; echo "resource,kind,project,name,generated $now" > results/compute.txt
for i in $(cat ~/Documents/dev/scripts/all-gcp-projects.txt) 
do echo "$i" 
    gcloud compute disks list --format json --project $i --quiet | ./dasel-tool -kind computedisk >> results/compute.txt 
    gcloud compute instances list --format json --project $i --quiet | ./dasel-tool -kind computeinstance >> results/compute.txt 
done 

now=$(date) ; echo "resource,kind,project,name,generated $now" > results/secrets.txt
for i in $(cat ~/Documents/dev/scripts/all-gcp-projects.txt) ; do echo "$i" ;gcloud secrets list --format=json --project $i --quiet | ./dasel-tool -kind secrets --fetchProjects >> results/secrets.txt ; done 

now=$(date) ; echo "resource,kind,project,name,generated $now" > results/bigtable.txt
for i in $(cat ~/Documents/dev/scripts/all-gcp-projects.txt) ; do echo "$i" ;gcloud bigtable instances list --format json --project $i --quiet | ./dasel-tool -kind bigtable >> results/bigtable.txt ; done 

```