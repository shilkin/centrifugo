CMAKE_MINIMUM_REQUIRED(VERSION 2.8.2)
PROJECT(centrifugo)

SET(CMAKE_MODULE_PATH ${CMAKE_MODULE_PATH} "${CMAKE_SOURCE_DIR}/cmake/Modules/")
INCLUDE(FindGo)

GoProjectConfigure(
    PKGNAME github.com/shilkin/centrifugo
)

GoBuild(
    TARGET centrifugo 
    PACKAGE github.com/shilkin/centrifugo
)

# No tests available for centrifugo
# GoTest(
#     PACKAGES
#         github.com/shilkin/centrifugo
# )

INSTALL(
    PROGRAMS ${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_NAME}
    DESTINATION bin COMPONENT runtime
)
