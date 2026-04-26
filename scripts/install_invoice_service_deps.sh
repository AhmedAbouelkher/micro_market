#!/bin/bash

# Invoice service name
INVOICE_SERVICE_NAME="invoice-service"
INVOICE_SERVICE_LIBS_PATH="$INVOICE_SERVICE_NAME/libs"

# check if we are in the root directory, if not move back to the root directory.
if [ "$(basename "$(pwd)")" != "micro_market" ]; then
    echo "We are not in the micro_market directory, moving back to the micro_market directory"
    cd ..
fi

install_git_submodules() {
    # Install the git submodules
    echo "Installing the git submodules"
    git submodule update --init --recursive

    # Check if the PDFGen submodule and httpserver exists, if not, throw an error
    if [ ! -d "$INVOICE_SERVICE_LIBS_PATH/PDFGen" ]; then
        echo "PDFGen submodule not found, throwing an error"
        exit 1
    fi

    if [ ! -d "$INVOICE_SERVICE_LIBS_PATH/httpserver" ]; then
        echo "httpserver submodule not found, throwing an error"
        exit 1
    fi
}

# if wget is not installed, install it (darwin and linux)
if ! command -v wget &> /dev/null; then
    if [ "$(uname)" == "Darwin" ]; then
        brew install wget
    elif [ "$(uname)" == "Linux" ]; then
        sudo apt-get install wget
    else
        echo "Unsupported platform"
        exit 1
    fi
fi

sqlite3_install() {
    # Check if sqlite3 in the libs/sqlite3 directory exists, if not, download it from (https://sqlite.org/2026/sqlite-amalgamation-3530000.zip) and extract it to the libs/sqlite3 directory
    if [ ! -f "$INVOICE_SERVICE_LIBS_PATH/sqlite3" ]; then
        echo "sqlite3 not found, downloading it"
        wget -O sqlite_amalgamation.zip https://sqlite.org/2026/sqlite-amalgamation-3530000.zip
        unzip sqlite_amalgamation.zip -d sqlite_amalgamation   
        mv sqlite_amalgamation/* $INVOICE_SERVICE_LIBS_PATH/sqlite3
    fi

    # Check if the libs/sqlite3 directory exists, if not, throw an error
    if [ ! -d "$INVOICE_SERVICE_LIBS_PATH/sqlite3" ]; then
        echo "sqlite3 directory not found, throwing an error"
        exit 1
    fi
}

# install hiredis and libuv in parallel
hiredis_install() {
    if [ -f "/usr/local/lib/libhiredis.a" ]; then
        echo "hiredis is already installed"
        return
    fi

    if [ ! -f "/usr/local/lib/libhiredis.a" ]; then
        echo "hiredis not found, downloading it"
        wget -O hiredis.zip https://github.com/redis/hiredis/archive/refs/heads/master.zip

        # if unzip is not installed, install it (darwin and linux)
        if ! command -v unzip &> /dev/null; then
            if [ "$(uname)" == "Darwin" ]; then
                brew install unzip
            elif [ "$(uname)" == "Linux" ]; then
                sudo apt-get install unzip
            else
                echo "Unsupported platform"
                exit 1
            fi
        fi

        unzip hiredis.zip
        cd hiredis-master
        make
        make install
        cd ..

        # check if hiredis is installed in the /usr/local/lib directory, if not throw an error
        if [ ! -f "/usr/local/lib/libhiredis.a" ]; then
            echo "hiredis failed to install"
            exit 1
        fi
    fi
}

libuv_install() {
    if [ -f "/usr/local/lib/libuv.a" ]; then
        echo "libuv is already installed"
        return
    fi

    if [ ! -f "/usr/local/lib/libuv.a" ]; then
        echo "libuv not found, downloading it"
        wget -O libuv.tar.gz https://dist.libuv.org/dist/v1.9.1/libuv-v1.9.1.tar.gz
        tar -xzf libuv.tar.gz
        cd libuv-v1.9.1
        sh autogen.sh
        ./configure --prefix=/usr/local
        make
        make install
        cd ..
  
        # check if libuv is installed in the /usr/local/lib directory, if not throw an error
        if [ ! -f "/usr/local/lib/libuv.a" ]; then
            echo "libuv failed to install"
            exit 1
        fi
    fi


}

clean_up() {
    echo "Cleaning up"
    rm -rf sqlite_amalgamation.zip
    rm -rf sqlite_amalgamation
    rm -rf hiredis.zip
    rm -rf hiredis-master
    rm -rf libuv.tar.gz
    rm -rf libuv-v1.9.1
}

install_deps() {
    install_git_submodules
    sqlite3_install
    hiredis_install
    libuv_install
    
    clean_up
}

install_deps
echo "Dependencies installed successfully"