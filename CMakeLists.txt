CMAKE_MINIMUM_REQUIRED(VERSION 2.8.2)
PROJECT(centrifugo)

MACRO(LOCATE_PATH VARIABLE FILE_NAME)
    IF(${VARIABLE}_TMP AND ${VARIABLE}_PATH)
        RETURN()
    ENDIF()

    FIND_PATH(${VARIABLE}_TMP NAMES ${FILE_NAME})

    IF(${VARIABLE}_TMP)
        SET(${VARIABLE}_PATH ${${VARIABLE}_TMP}/${FILE_NAME})
        MESSAGE(STATUS "${FILE_NAME} at: ${${VARIABLE}_PATH}")
    ELSE()
        MESSAGE(FATAL_ERROR "${FILE_NAME} not found.")
    ENDIF()
ENDMACRO()

LOCATE_PATH(GO_MODULE_CMAKE share/sputnik/gopack/cmake)
SET(CMAKE_MODULE_PATH ${CMAKE_MODULE_PATH} ${GO_MODULE_CMAKE_PATH})
INCLUDE(FindGo2)

GoProjectConfigure(
    PKGNAME github.com/shilkin/centrifugo
)

GoDepends(
    PACKAGES gitlab.srv.pv.km/mailpaas/ttconnector-framework
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
