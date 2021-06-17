import glob
import os,sys
filelist = glob.glob("/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/examples/result_task2_wrf_run-at-20210608-150432/count*.md")
rankfile = "/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/examples/rank"
rankfile1 = "/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/examples/rank1"

def swith_node(a,b):
    inp = open(rankfile,'r')
    out = open(rankfile1,'w')
    for enum,line in enumerate(inp):
        if enum==a:
            prev = line
            if enum ==b:
                out.write(out)
                out.write(prev)
        else:
            out.write(prev)
    os.system("mv /home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/examples/rank")

def get_result():
    res_map= 1
    map_dict = dict(map(),int)
    # for item in filelist:
    if True: # test
        item = filelist[0] # test
        temp_file = open(item,'r')
        alltoallv_map_array = {}
        map_set = temp_file.read().replace("Send datatype size: 1","").replace("Recv datatype size: 1","").replace("Comm size: 160","").replace("Send counts","").replace("Recv counts","").replace("\n","").split(" ")[0:-1]
        for i in range(len(map_set)/2):
            if map_set[i] != 0:

        # get the pattern communicate.
        temp_file.close()

def parseSLURM(string):
    """Return a host list from a SLURM string"""
    # Use scontrol utility to get the hosts list
    import subprocess, os
    hostsstr = subprocess.check_output(["scontrol", "show", "hostnames", string])
    if sys.version_info.major > 2:
        hostsstr = hostsstr.decode()
    # Split using endline
    hosts = hostsstr.split(os.linesep)
    # Take out last empty host
    hosts = filter(None, hosts)
    # Create the desired pair of host and number of hosts
    hosts = [(host, 1) for host in hosts]
    return hosts

def get_nodeinfo():
    try:
       return parseSLURM(os.environ["SLURM_NODELIST"])
    except:
       return (("nia0001","1"),("nia0002","1"),("nia0003","1"),("nia0004","1"))

def has_rank(str1):
    with open('rank','r') as myfile:
     if str1 in myfile.read():
        return False
     else:
        return True

def write_rank(nodelist):
    with open("rank","w") as file:
        for i in range(40):
            file.write("rank 0="+nodelist[0][0]+ " slot=1\n")
        for i in range(40):
            file.write("rank 1="+nodelist[1][0]+ " slot=1\n")
        for i in range(40):
            file.write("rank 2="+nodelist[2][0]+ " slot=1\n")
        for i in range(40):
            file.write("rank 3="+nodelist[3][0]+ " slot=1\n")

#print(get_nodeinfo()[0][0])
if has_rank(get_nodeinfo()[0][0]):
    write_rank(get_nodeinfo())
map_dict = sorted(get_result())

for i in range(10):
    