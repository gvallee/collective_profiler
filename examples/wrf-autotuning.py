import glob
import os,sys
import argparse

parser = argparse.ArgumentParser(description='Date.')
parser.add_argument('--date',  type=str,
                    help='Job Name')
args = parser.parse_args()
sb=args.date

filelist = glob.glob("/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/examples/results_task2_wrf/run-at-"+ sb + "/*.md")
rankfile = "/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/examples/rank"

def swith_node(a,b):
    inp = open(rankfile,'r')
    temp_inp = inp.read()
    inp.close()
    out = open(rankfile,'w')
    out.write(temp_inp.replace(" "+str(a[0]+2)+"=nia","sb").replace(" "+str(b[0])+"=nia"," "+str(a[0]+2)+"=nia").replace("sb"," "+str(b[0])+"=nia"))
    out.close()

def get_result():
    res_map= 1
    map_dict = {}
    for item in filelist:
    # if True: # test
        # item = filelist[0] # test
        temp_file = open(item,'r')
        alltoallv_map_array = {}
        map_set = temp_file.read().replace("Send datatype size: 1","").replace("Recv datatype size: 1","").replace("Comm size: 160","").replace("Send counts","").replace("Recv counts","").replace("\n","").split(" ")[0:-1]
        for i in range(int(len(map_set)/2)):
            if map_set[i] != '0':
                try:
                    map_dict[(i%160,map_set.index(map_set[i],int(len(map_set)/2))%160)]+=1
                except:
                    d = {(i%160,map_set.index(map_set[i],int(len(map_set)/2))%160):1}
                    map_dict.update(d)
        # get the pattern communicate.
        # print(sorted(map_dict))
        temp_file.close()
    return (sorted(map_dict)[0:10])

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
    with open(rankfile,'r') as myfile:
     if str1 in myfile.read():
        return False
     else:
        return True

def write_rank(nodelist):
    with open(rankfile,"w") as file:
        for i in range(40):
            file.write("rank "+str(i)+"="+nodelist[0][0]+ " slot=1\n")
        for i in range(40):
            file.write("rank "+str(i+40)+"="+nodelist[1][0]+ " slot=1\n")
        for i in range(40):
            file.write("rank "+str(i+80)+"="+nodelist[2][0]+ " slot=1\n")
        for i in range(40):
            file.write("rank "+str(i+120)+"="+nodelist[3][0]+ " slot=1\n")


if has_rank(get_nodeinfo()[0][0]):
    write_rank(get_nodeinfo())
temp_result=get_result()
for item in temp_result:
    swith_node(*zip(item))
