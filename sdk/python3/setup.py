from setuptools import setup, find_packages

setup(
    name='nmidsdk',
    version='0.1.1',
    packages=find_packages(),
    install_requires=[
        'requests',
    ],
    author='hughnian',
    author_email='hughnian@163.com',
    description='nmid micro service framework python3 sdk',
    long_description=open('README.md').read(),
    long_description_content_type='text/markdown',
    url='https://github.com/hughnian/nmid',
    classifiers=[
        'Programming Language :: Python :: 3',
        'License :: OSI Approved :: MIT License',
        'Operating System :: OS Independent',
    ],
)