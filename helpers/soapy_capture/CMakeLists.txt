cmake_minimum_required(VERSION 3.16)
project(gpsdr_soapy LANGUAGES CXX)

add_executable(gpsdr-soapy main.cpp)
target_compile_features(gpsdr-soapy PRIVATE cxx_std_17)
target_link_libraries(gpsdr-soapy PRIVATE ${CMAKE_DL_LIBS})

install(TARGETS gpsdr-soapy RUNTIME DESTINATION bin)
