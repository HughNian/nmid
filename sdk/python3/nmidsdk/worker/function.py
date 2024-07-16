from typing import Callable, Union
from nmidsdk.worker.response import Response

JobFunc = Callable[[Response], Union[bytes, ConnectionError]]

class Function:
    def __init__(self, jf: JobFunc, fname: str):
        self.func = jf
        self.func_name = fname