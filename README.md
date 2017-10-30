# hmail-log-parser
a program to parse hMail server log hmailserver_yyyy-MM-dd.log
there are three types of command parameters
    1) impute parameter; the program uses this to know where to get data (-d and -c)
    2) output parameters; used to decide what to produce and where (-o and -summary)
    3) filters; these are the parameters used to decide which record to keep and which to skip (the rest)

# Usage
hmail-log-parser.exe [-d <logDate>]  -c <configFilePath> [-i <remoteIP>] [-s <sessionID> || -session] [-server || -client] [-o <outputFilePath>] [-m <searchString>] [-summary]

-d <logDate>
    this is the date of the log file to parse in the format yyyy-MM-dd (defaults to the current date)

-c <configFilePath>
    specified the path to the config file. when not specified,the program looks for a file named conf in the same directory as the executable

-i <remoteIP>
    selects only log records with the specified remote IP

-s <sessionID>
    selects only log records with the specified session ID (integer). cannot be used with -session

-server
    when included, only records from the SMTP server (SMTPD) is selected. cannot be used with -client

-client
    when included, only records from the SMTP client (SMTPC) is selected. cannot be used with -server

-session
    when included, it returns not only records that matches the filters but the entire session. cannot be used with the -s

-m <searchString>
    returns only records that contains the search string in the message field. this can be specified more than once to search for multiple strings. the behavior depends on whether -session is specified or not. if -session is true, it will search the entire session for the strings and will return any session that contains all the strings. if session is not specified, each record returned must contain all the strings.

-o <outputFilePath>
    when specified, the result is sent to the file instead of the console. the path must not be that of an existing file

-summary
    outputs a summary of the log file parsed. these includes the number of records, the number of records for each remote ip, the number of records for each hour and the total number of sessions. this will be displaced in addition to the results of the filter parameters

# Config File
The config file is a json file used to specify the location of the log file. it contains two fields
    "LogDir"
        the directory where the active log files are located. this is will the where hMailServer keeps its logs
    
    "LogRepoDir"
        the directory where log files are moved e.g. to free space on the drive the server writes it logs
    
#Config file example

{
    "LogDir": "data",
    "LogRepoDir": "bckup"
}

the config file above show that the log files are in a data directory relative to the executable while the backup location is in a bckup directory also relative to the executable

#Example

    This example searches the current log file in directories specified by a config file conf for sessions which has messages that contains both a@example.com and b@example.com, the only log entries considered are the ones from the mail server (SMTPD), which means SMTPC logs are not considered. the sumamry of the parses log is also included in the result which is sent to a file named output.log

    hmail-log-parser.exe -i 10.15.0.1 -server -session -m a@example.com -m b@example.com -o output.log -summary

    