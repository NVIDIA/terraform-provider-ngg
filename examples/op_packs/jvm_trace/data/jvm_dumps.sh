#!/bin/bash

# required: regex to identify the java process uniquely
# aws cli, s3 permissions to dump jvm info
# the user needs to have permissions to execute jcmd, jps, jmap, jstat, jstack

JVM_PROCESS_REGEX=${JVM_PROCESS_REGEX:-$1}
S3_BUCKET=${S3_BUCKET:-$2}

if [ -z "${JVM_PROCESS_REGEX}" ] || [ -z "${S3_BUCKET}" ]; then
    printf -- "JVM_PROCESS_REGEX and S3_BUCKET are required inputs exiting.\n"
    exit 127
fi

pid=$(jps | grep "${JVM_PROCESS_REGEX}" | awk '{print $1}')
timestamp=$(date +%Y%m%d%H%M%S)
#killall5
# take heap dump using

heap_dump(){
    if command -v jcmd &> /dev/null
    then
        printf -- "using jcmd to take heap dump.\n"
        jcmd_dump="/tmp/heap_dump_${pid}_${timestamp}"
        jcmd ${pid} GC.heap_dump ${jcmd_dump} &> /dev/null
        aws s3 cp ${jcmd_dump} s3://${S3_BUCKET}/java-dumps-${timestamp}/heap_dump.hprof &> /dev/null
        printf -- "heap dump upload destination: s3://${S3_BUCKET}/java-dumps-${timestamp}/heap_dump.hprof\n"
        return
    elif command -v jmap &> /dev/null
    then
        printf -- "using jmap to take heap dump.\n"
        jmap_dump="/tmp/heap_dump_${pid}_${timestamp}"
        jmap -dump:live,format=b,file=${jmap_dump} $pid &> /dev/null
        aws s3 cp ${jmap_dump} s3://${S3_BUCKET}/java-dumps-${timestamp}/heap_dump.hprof &> /dev/null
        printf -- "heap dump upload destination: s3://${S3_BUCKET}/java-dumps-${timestamp}/heap_dump.hprof\n"
        return
    else
        printf -- "jcmd or jmap not found, skipping heap dump.\n"
    fi
}

thread_dump(){
    if command -v jstack &> /dev/null
    then
        printf -- "using jstack to take thread dump.\n"
        thread_dump="/tmp/thread_dump_${pid}_${timestamp}"
        jstack -F ${pid} > ${thread_dump}
        aws s3 cp ${thread_dump} s3://${S3_BUCKET}/java-dumps-${timestamp}/thread_info.txt &> /dev/null
        printf -- "thread dump upload destination: s3://${S3_BUCKET}/java-dumps-${timestamp}/thread_info.txt\n"
        return
    else
        printf -- "jstack not found skipping thread dump.\n"
    fi
}

gc_info(){
    if command -v jstat &> /dev/null
    then
        printf -- "using jstat to get gc info.\n"
        gc_dump="/tmp/gc_info_${pid}_${timestamp}"
        jstat -gc $pid 1000 5 > ${gc_dump}
        aws s3 cp ${gc_dump} s3://${S3_BUCKET}/java-dumps-${timestamp}/gc_info.txt &> /dev/null
        printf -- "garbage collector info upload destination: s3://${S3_BUCKET}/java-dumps-${timestamp}/gc_info.txt\n"
        return

    else
        echo "jstack not found skipping gc info collection.\n"
    fi
}

heap_info(){
    if command -v jstat &> /dev/null
    then
        printf -- "using jmap to get heap stats.\n"
        heap_stats="/tmp/heap_info_${pid}_${timestamp}"
        jmap -heap $pid > ${heap_stats}
        aws s3 cp ${heap_stats} s3://${S3_BUCKET}/java-dumps-${timestamp}/heap_stats.txt &> /dev/null
        printf -- "heap stats upload destination: s3://${S3_BUCKET}/java-dumps-${timestamp}/heap_stats.txt\n"
        return
    else
        echo "jmap not found skipping heap info collection.\n"
    fi
}

detect_deadlock() {
    jstack -F -m ${pid} | grep "No deadlocks found" &> /dev/null
    if [ $? -eq 1 ]; then
        printf -- "Deadlocks were detected in jstack output.\n"
    else
        printf -- "No deadlocks detected in jstack output.\n"
    fi
}

heap_dump
heap_info
thread_dump
gc_info
detect_deadlock
printf -- "The s3 location for the jvm dumps is ${S3_BUCKET}/java-dumps-${timestamp}/.\n"

# Don't kill the pod for now.
#killall5

